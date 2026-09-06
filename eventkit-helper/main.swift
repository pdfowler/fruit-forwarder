import EventKit
import Foundation

private func trace(_ stage: String) {
    if ProcessInfo.processInfo.environment["BRIDGE_EVENTKIT_TRACE"] == "1" {
        FileHandle.standardError.write(Data("eventkit-stage: \(stage)\n".utf8))
    }
}

private struct WireItem: Codable {
    var uid: String = ""
    var summary: String = ""
    var status: String = "needs_action"
    var description: String = ""
    var due: String = ""
    var completed: String = ""
    var modified: String = ""

    enum CodingKeys: String, CodingKey {
        case uid, summary, status, description, due, completed, modified
    }

    init(
        uid: String = "",
        summary: String = "",
        status: String = "needs_action",
        description: String = "",
        due: String = "",
        completed: String = "",
        modified: String = ""
    ) {
        self.uid = uid
        self.summary = summary
        self.status = status
        self.description = description
        self.due = due
        self.completed = completed
        self.modified = modified
    }

    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        uid = try values.decodeIfPresent(String.self, forKey: .uid) ?? ""
        summary = try values.decodeIfPresent(String.self, forKey: .summary) ?? ""
        status = try values.decodeIfPresent(String.self, forKey: .status) ?? "needs_action"
        description = try values.decodeIfPresent(String.self, forKey: .description) ?? ""
        due = try values.decodeIfPresent(String.self, forKey: .due) ?? ""
        completed = try values.decodeIfPresent(String.self, forKey: .completed) ?? ""
        modified = try values.decodeIfPresent(String.self, forKey: .modified) ?? ""
    }
}

private struct WireList: Codable {
    let id: String
    let name: String
    let source: String
    let readOnly: Bool
    var items: [WireItem]

    enum CodingKeys: String, CodingKey {
        case id, name, source, items
        case readOnly = "read_only"
    }
}

private struct Request: Codable {
    let action: String
    var listIDs: [String] = []
    var listID: String = ""
    var item: WireItem = WireItem()

    enum CodingKeys: String, CodingKey {
        case action, item
        case listIDs = "list_ids"
        case listID = "list_id"
    }

    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        action = try values.decode(String.self, forKey: .action)
        listIDs = try values.decodeIfPresent([String].self, forKey: .listIDs) ?? []
        listID = try values.decodeIfPresent(String.self, forKey: .listID) ?? ""
        item = try values.decodeIfPresent(WireItem.self, forKey: .item) ?? WireItem()
    }
}

private struct Response: Codable {
    var lists: [WireList]?
    var item: WireItem?
}

private enum HelperError: LocalizedError {
    case accessDenied
    case invalidRequest(String)
    case listUnavailable(String)
    case listReadOnly(String)
    case reminderUnavailable(String)
    case reminderOutOfScope

    var errorDescription: String? {
        switch self {
        case .accessDenied:
            return "Reminders access was not granted"
        case .invalidRequest(let detail):
            return "Invalid request: \(detail)"
        case .listUnavailable(let id):
            return "Reminder list is unavailable: \(id)"
        case .listReadOnly(let id):
            return "Reminder list is read-only: \(id)"
        case .reminderUnavailable(let id):
            return "Reminder is unavailable: \(id)"
        case .reminderOutOfScope:
            return "Reminder does not belong to the requested list"
        }
    }
}

private final class EventKitService {
    private let store = EKEventStore()
    private let iso8601 = ISO8601DateFormatter()
    private let dateOnly: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    func requestAccess() async throws {
        let status = EKEventStore.authorizationStatus(for: .reminder)
        trace("authorization-status-\(status.rawValue)")
        switch status {
        case .fullAccess:
            return
        case .denied, .restricted:
            throw HelperError.accessDenied
        default:
            break
        }
        let granted = try await store.requestFullAccessToReminders()
        guard granted else { throw HelperError.accessDenied }
    }

    func lists() -> [WireList] {
        store.calendars(for: .reminder)
            .map {
                WireList(
                    id: $0.calendarIdentifier,
                    name: $0.title,
                    source: $0.source.title,
                    readOnly: !$0.allowsContentModifications,
                    items: []
                )
            }
            .sorted { ($0.source, $0.name, $0.id) < ($1.source, $1.name, $1.id) }
    }

    func snapshot(listIDs: [String]) async throws -> [WireList] {
		guard !listIDs.isEmpty else { return [] }
        let calendars = try listIDs.map { try calendar(id: $0, writable: false) }
        let reminders = await fetchReminders(calendars: calendars)
        let grouped = Dictionary(grouping: reminders, by: { $0.calendar.calendarIdentifier })
        return calendars.map { calendar in
            var list = WireList(
                id: calendar.calendarIdentifier,
                name: calendar.title,
                source: calendar.source.title,
                readOnly: !calendar.allowsContentModifications,
                items: (grouped[calendar.calendarIdentifier] ?? []).map(wireItem)
            )
            list.items.sort { ($0.status, $0.summary, $0.uid) < ($1.status, $1.summary, $1.uid) }
            return list
        }
    }

    func create(listID: String, item: WireItem) throws -> WireItem {
        guard !item.summary.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            throw HelperError.invalidRequest("summary is required")
        }
        let reminder = EKReminder(eventStore: store)
        reminder.calendar = try calendar(id: listID, writable: true)
        reminder.title = item.summary
        reminder.notes = item.description.isEmpty ? nil : item.description
        reminder.dueDateComponents = try dueComponents(item.due)
        reminder.isCompleted = item.status == "completed"
        try store.save(reminder, commit: true)
        return wireItem(reminder)
    }

    func update(listID: String, item: WireItem) throws -> WireItem {
        guard !item.summary.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            throw HelperError.invalidRequest("summary is required")
        }
        let reminder = try scopedReminder(id: item.uid, listID: listID)
        reminder.title = item.summary
        reminder.notes = item.description.isEmpty ? nil : item.description
        reminder.dueDateComponents = try dueComponents(item.due)
        reminder.isCompleted = item.status == "completed"
        try store.save(reminder, commit: true)
        return wireItem(reminder)
    }

    func setCompleted(listID: String, itemID: String, completed: Bool) throws -> WireItem {
        let reminder = try scopedReminder(id: itemID, listID: listID)
        reminder.isCompleted = completed
        try store.save(reminder, commit: true)
        return wireItem(reminder)
    }

    private func calendar(id: String, writable: Bool) throws -> EKCalendar {
        guard let calendar = store.calendar(withIdentifier: id),
              calendar.allowedEntityTypes.contains(.reminder) else {
            throw HelperError.listUnavailable(id)
        }
        if writable && !calendar.allowsContentModifications {
            throw HelperError.listReadOnly(id)
        }
        return calendar
    }

    private func scopedReminder(id: String, listID: String) throws -> EKReminder {
        guard let reminder = store.calendarItem(withIdentifier: id) as? EKReminder else {
            throw HelperError.reminderUnavailable(id)
        }
        guard reminder.calendar.calendarIdentifier == listID else {
            throw HelperError.reminderOutOfScope
        }
        _ = try calendar(id: listID, writable: true)
        return reminder
    }

    private func fetchReminders(calendars: [EKCalendar]) async -> [EKReminder] {
        let predicate = store.predicateForReminders(in: calendars)
        return await withCheckedContinuation { continuation in
            store.fetchReminders(matching: predicate) { reminders in
                continuation.resume(returning: reminders ?? [])
            }
        }
    }

    private func wireItem(_ reminder: EKReminder) -> WireItem {
        WireItem(
            uid: reminder.calendarItemIdentifier,
            summary: reminder.title ?? "",
            status: reminder.isCompleted ? "completed" : "needs_action",
            description: reminder.notes ?? "",
            due: wireDue(reminder.dueDateComponents),
            completed: wireDate(reminder.completionDate),
            modified: wireDate(reminder.lastModifiedDate)
        )
    }

    private func wireDue(_ components: DateComponents?) -> String {
        guard let components else { return "" }
        if components.hour == nil && components.minute == nil && components.second == nil,
           let date = components.calendar?.date(from: components) ?? Calendar.current.date(from: components) {
            return dateOnly.string(from: date)
        }
        guard let date = components.calendar?.date(from: components) ?? Calendar.current.date(from: components) else {
            return ""
        }
        return iso8601.string(from: date)
    }

    private func wireDate(_ date: Date?) -> String {
        date.map(iso8601.string) ?? ""
    }

    private func dueComponents(_ value: String) throws -> DateComponents? {
        if value.isEmpty { return nil }
        if value.count == 10, let date = dateOnly.date(from: value) {
            var components = Calendar.current.dateComponents([.year, .month, .day], from: date)
            components.calendar = Calendar.current
            components.timeZone = TimeZone.current
            return components
        }
        guard let date = iso8601.date(from: value) else {
            throw HelperError.invalidRequest("due must be YYYY-MM-DD or RFC3339")
        }
        var components = Calendar.current.dateComponents(
            [.year, .month, .day, .hour, .minute, .second, .timeZone], from: date
        )
        components.calendar = Calendar.current
        return components
    }
}

@main
private struct Main {
    static func main() async {
        do {
            trace("reading-request")
            let input = FileHandle.standardInput.readDataToEndOfFile()
            let request = try JSONDecoder().decode(Request.self, from: input)
            trace("request-decoded")
            let service = EventKitService()
            trace("event-store-created")
            try await service.requestAccess()
            trace("access-granted")
            let response: Response
            switch request.action {
            case "lists":
                response = Response(lists: service.lists(), item: nil)
            case "snapshot":
                response = Response(lists: try await service.snapshot(listIDs: request.listIDs), item: nil)
            case "create":
                response = Response(lists: nil, item: try service.create(listID: request.listID, item: request.item))
            case "update":
                response = Response(lists: nil, item: try service.update(listID: request.listID, item: request.item))
            case "complete":
                response = Response(lists: nil, item: try service.setCompleted(listID: request.listID, itemID: request.item.uid, completed: true))
            case "reopen":
                response = Response(lists: nil, item: try service.setCompleted(listID: request.listID, itemID: request.item.uid, completed: false))
            default:
                throw HelperError.invalidRequest("unsupported action")
            }
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.sortedKeys]
            FileHandle.standardOutput.write(try encoder.encode(response))
        } catch {
            FileHandle.standardError.write(Data("eventkit-helper: \(error.localizedDescription)\n".utf8))
            Foundation.exit(1)
        }
    }
}
