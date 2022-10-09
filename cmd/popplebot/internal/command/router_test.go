package command

import "testing"

func TestRoute(t *testing.T) {
	type result struct {
		typecheck func(a ArgParser)
		remainder string
	}

	tests := []struct {
		input string
		want  result
	}{
		{
			input: "popple announce on",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*SetAnnounceArgs) },
				remainder: " on",
			},
		},
		{
			input: "popple karma potato",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*CheckKarmaArgs) },
				remainder: " potato",
			},
		},
		{
			input: "popple top 10",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*LeaderboardArgs) },
				remainder: " 10",
			},
		},
		{
			input: "popple bot",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*LoserboardArgs) },
				remainder: "",
			},
		},
		{
			input: "some text",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*ChangeKarmaArgs) },
				remainder: "some text",
			},
		},
		{
			input: "popple    announce off",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*SetAnnounceArgs) },
				remainder: " off",
			},
		},
		{
			input: "popple          karma tomato",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*CheckKarmaArgs) },
				remainder: " tomato",
			},
		},
		{
			input: "popple     bot",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*LoserboardArgs) },
				remainder: "",
			},
		},
		{
			input: "popple                 top 3",
			want: result{
				typecheck: func(a ArgParser) { _ = a.(*LeaderboardArgs) },
				remainder: " 3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			router := NewRouter("popple")

			args, rem := router.Route(tt.input)

			tt.want.typecheck(args)
			if rem != tt.want.remainder {
				t.Errorf("want remainder %q, got remainder %q", tt.want.remainder, rem)
			}
		})
	}
}
