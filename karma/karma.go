package karma

func Bump(current map[string]int64, increment map[string]int64) (bumped map[string]int64) {
	bumped = make(map[string]int64)
	for name, incr := range increment {
		if incr == 0 {
			continue
		}
		bumped[name] = current[name] + incr
	}
	return bumped
}
