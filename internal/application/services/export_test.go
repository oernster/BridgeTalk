package services

// Wait blocks until the making under way ends, so a test reads what making did rather than
// racing it.
func (m *MakingService) Wait() { m.endRun(false) }

// AuditionsWaiting counts the auditions waiting for the model's turn, so a test knows one is queued
// before it lets the line under way finish.
func (m *MakingService) AuditionsWaiting() int { return m.turn.waiting() }
