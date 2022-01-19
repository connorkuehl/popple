package karma

import "github.com/connorkuehl/popple/internal/parse"

func Parse(text string) map[string]int64 {
	subjects := parse.Subjects(text)
	levels := make(map[string]int64)

	for _, s := range subjects {
		levels[s.Name] += s.Karma
	}

	return levels
}
