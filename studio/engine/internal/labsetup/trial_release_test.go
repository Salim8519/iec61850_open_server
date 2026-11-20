//go:build trial

package labsetup

// callRelease is Release for the trial harness. Before change 0221 Release returned nothing; the
// "before" numbers were taken with `s.Release(); return nil` here.
func callRelease(s *Service) any {
	if err := s.Release(); err != nil {
		return err
	}
	return nil
}
