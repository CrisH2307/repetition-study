package cite

import "testing"

const src = "The rate of the race is the sum\nof both rates, so T ~ Exp(λ1 + λ2).\n\n" +
	"$$\n e^{-\\lambda t} \n$$\n" +
	"and also \\[P(X=k)\\] for each k."

func TestVerify(t *testing.T) {
	for _, tc := range []struct {
		name, quote string
		want        bool
	}{
		{"spans a line break", "is the sum of both rates", true},
		{"extra whitespace in quote", "  the   sum\n\nof both  rates ", true},
		{"one word", "rates", false},
		{"short phrase under the minimum", "sum of both", false},
		{"case change", "The Rate of the race is the sum", false},
		{"not in source", "the rate of the race is the product", false},
		{"empty", "  \n ", false},
		{"whole display formula, inner", "e^{-\\lambda t}", true},
		{"whole display formula, delimited", "$$ e^{-\\lambda t} $$", true},
		{"whole bracket formula", "P(X=k)", true},
		{"garbled math fragment", "^{-\\lambda", false},
		{"garbled math not in source", "e^{-λ t} = λ1λ2 + ( / )", false},
		{"short math fragment of a sentence", "Exp(λ1 + λ2)", false},
	} {
		if got := Verify(src, tc.quote); got != tc.want {
			t.Errorf("%s: Verify(%q) = %v, want %v", tc.name, tc.quote, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize(" a\n\tb  c \n"); got != "a b c" {
		t.Fatalf("got %q", got)
	}
}
