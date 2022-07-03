package support

import "testing"

func TestStudlyCase(t *testing.T) {
	a := "xx_yy"
	b := StudlyCase(a)
	expected := "XxYy"
	if b != expected {
		t.Errorf("Expected the `StudlyCase` of %s to be %s but instead got %s !", a, expected, b)
	}
}

func TestCamelCase(t *testing.T) {
	a := "xx_yy"
	b := CamelCase(a)
	expected := "xxYy"
	if b != expected {
		t.Errorf("Expected the `CamelCase` of %s to be %s but instead got %s !", a, expected, b)
	}

	a_2 := "x"
	b_2 := CamelCase(a_2)
	expected_2 := "x"
	if b_2 != expected_2 {
		t.Errorf("Expected the `CamelCase` of %s to be %s but instead got %s !", a_2, expected_2, b_2)
	}
}

func TestSnakeCase(t *testing.T) {
	a := "XxYy"
	b := SnakeCase(a)
	expected := "xx_yy"
	if b != expected {
		t.Errorf("Expected the `SnakeCase` of %s to be %s but instead got %s !", a, expected, b)
	}

	a = "XxYY"
	b = SnakeCase(a)
	expected = "xx_yy"
	if b != expected {
		t.Errorf("Expected the `SnakeCase` of %s to be %s but instead got %s !", a, expected, b)
	}

	a = "ID"
	b = SnakeCase(a)
	expected = "id"
	if b != expected {
		t.Errorf("Expected the `SnakeCase` of %s to be %s but instead got %s !", a, expected, b)
	}
}
