package service

// Cache exposes the underlying memory cache to observers (read-only use)
func (s *PresenceService) Cache() interface{ Size() int } { return s.cache }
