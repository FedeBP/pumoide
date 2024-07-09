package constants

const (
	// General constants
	EmptyString = ""

	// Action constants
	Action                 = "action"
	ActionExport           = "export"
	ActionImport           = "import"
	ActionAddRequest       = "addRequest"
	ActionUpdateCollection = "updateCollection"
	ActionDeleteCollection = "deleteCollection"
	ActionDeleteRequest    = "deleteRequest"
	ActionEntry            = "getEntry"
	ActionDeleteEntry      = "deleteEntry"
	ActionClearHistory     = "clearHistory"

	// ID constants
	CollectionID = "collectionId"
	RequestID    = "requestId"
	ID           = "id"

	// Other constants
	Env = "env"

	// Success messages
	CollectionDeletedSuccess  = "Collection deleted successfully"
	RequestDeletedSuccess     = "Request deleted successfully"
	EnvironmentDeletedSuccess = "Environment deleted successfully"
)
