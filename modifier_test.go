package main

import "testing"

func TestParseModifierEmpty(t *testing.T) {
	actual := ParseModifier("")
	if actual != nil {
		t.Fatalf("Expected nil Modifier when parsing empty string")
	}
}

func TestNoKarma(t *testing.T) {
	actual := ParseModifier("Poe")
	if actual != nil {
		t.Fatalf("Expected nil Modifier when parsing string with only name")
	}
}

func TestNoNameButValidKarma(t *testing.T) {
	actual := ParseModifier("++")
	if actual != nil {
		t.Fatalf("Expected nil Modifier when parsing string with no name")
	}
}

func TestSpaceBetweenNameAndKarma(t *testing.T) {
	actual := ParseModifier("Poe ++")
	if actual != nil {
		t.Fatalf("Expected nil Modifier with space separating name and karma")
	}
}

func TestNetZeroIsNil(t *testing.T) {
	actual := ParseModifier("Poe++--++--")
	if actual != nil {
		t.Fatalf("Expected nil Modifier with a net zero operation")
	}
}

func TestSimpleIncrement(t *testing.T) {
	actual := ParseModifier("Poe++")
	if actual == nil {
		t.Fatalf("Did not expect nil Modifier with valid input")
	}

	if actual.Name != "Poe" || actual.NetKarma != 1 {
		t.Fatalf("Expected Modifier{%s, %d}, got Modifier{%s, %d}", "Poe", 1, actual.Name, actual.NetKarma)
	}
}

func TestSimpleDecrement(t *testing.T) {
	actual := ParseModifier("Poe--")
	if actual == nil {
		t.Fatalf("Did not expect nil Modifier with valid input")
	}

	if actual.Name != "Poe" || actual.NetKarma != -1 {
		t.Fatalf("Expected Modifier{%s, %d}, got Modifier{%s, %d}", "Poe", -1, actual.Name, actual.NetKarma)
	}
}

func TestParseModifierWithIncrementsAndDecrements(t *testing.T) {
	actual := ParseModifier("Poe----++")
	if actual == nil {
		t.Fatalf("Did not expect nil Modifier with valid input")
	}

	if actual.Name != "Poe" || actual.NetKarma != -1 {
		t.Fatalf("Expected Modifier{%s, %d}, got Modifier{%s, %d}", "Poe", -1, actual.Name, actual.NetKarma)
	}
}

func TestParseModifierWithIncompleteOperator(t *testing.T) {
	actual := ParseModifier("Poe-")
	if actual != nil {
		t.Fatalf("Expected nil, got Modifier{%s, %d}", actual.Name, actual.NetKarma)
	}
}

func TestSomeLegitimateInputWithIncompleteOperator(t *testing.T) {
	actual := ParseModifier("Poe++-")
	if actual == nil {
		t.Fatalf("Did not expect nil Modifier with valid input")
	}

	if actual.Name != "Poe" || actual.NetKarma != 1 {
		t.Fatalf("Expected Modifier{%s, %d}, got Modifier{%s, %d}", "Poe", 1, actual.Name, actual.NetKarma)
	}
}

func TestParseModifierReturnsWithFirstModifierIgnoringRest(t *testing.T) {
	actual := ParseModifier("Poe++ThePotatoPirate-32-42-3423")
	if actual == nil {
		t.Fatalf("Did not expect nil Modifier with valid input")
	}

	if actual.Name != "Poe" || actual.NetKarma != 1 {
		t.Fatalf("Expected Modifier{%s, %d}, got Modifier{%s, %d}", "Poe", 1, actual.Name, actual.NetKarma)
	}
}

func TestParseModifierIsAtSymbolAware(t *testing.T) {
	actual := ParseModifier("@Poe++")
	if actual == nil {
		t.Fatalf("Did not expect nil Modifier with valid input")
	}

	if actual.Name != "Poe" || actual.NetKarma != 1 {
		t.Fatalf("Expected Modifier{%s, %d}, got Modifier{%s, %d}", "Poe", 1, actual.Name, actual.NetKarma)
	}
}
