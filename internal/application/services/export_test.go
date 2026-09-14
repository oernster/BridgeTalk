package services

// Wait blocks until the making under way ends, so a test reads what making did rather than
// racing it.
func (m *MakingService) Wait() { m.endRun(false) }
