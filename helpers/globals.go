package helpers

// This is nasty, but we need a way to reference globals outside of the main package
var TouchpanelStatusMap map[string]TouchpanelStatus // Global map of TouchpanelStatus to allow for status updates
var UpdateChannel chan TouchpanelStatus

var ValidationStatus map[string]TouchpanelStatus
var ValidationChannel chan TouchpanelStatus
