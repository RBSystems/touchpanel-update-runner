package helpers

// ChannelUpdater exists to avoid concurrent map write errors
func ChannelUpdater() {
	for true { // Loop forever
		touchpanelToUpdate := <-UpdateChannel                             // Watch for new things in the UpdateChannel
		TouchpanelStatusMap[touchpanelToUpdate.UUID] = touchpanelToUpdate // Add new things to the map that is queried when you ask for touchpanel status
	}
}

// ValidateHelper exists to avoid concurrent map write errors
func ValidateHelper() {
	for true { // Loop forever
		toAdd := <-ValidationChannel            // Watch for new things in the ValidationChannel
		ValidationStatus[toAdd.Address] = toAdd // Add new things to the map that is queried when you ask for touchpanel status
	}
}
