package helpers

// ChannelUpdater just updates the channel so we can get around concurrent map write issues
func ChannelUpdater() {
	for true {
		tpToUpdate := <-UpdateChannel
		TouchpanelStatusMap[tpToUpdate.UUID] = tpToUpdate
	}
}
