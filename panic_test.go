package main

import (
	"slices"
	"testing"
)

func TestNilOrBadSlice(t *testing.T) {

	_, err := packLine(nil, false)

	if err == nil {

		t.Errorf("Expected error")
	}

	_, err = packLine(make([]byte, 1), false)

	if err == nil {

		t.Errorf("Expected error")
	}

}

func TestAllZeroSlice(t *testing.T) {

	expected := make([]byte, 1)
	got, err := packLine(make([]byte, 8), false)

	if err != nil {

		t.Errorf("Expected a nil error")
	}

	if slices.Compare(got, expected) != 0 {

		t.Errorf("Unpexted return slice")
	}
}

func TestRunLength(t *testing.T) {

	tests := make([][]byte, 0)
	expected := make([][]byte, 0)

	tests = append(tests, []byte{0, 0, 0, 0, 0, 1, 1, 1})
	expected = append(expected, []byte{0x07, 0xf3, 1})

	tests = append(tests, []byte{8, 0, 0, 0, 0, 0, 0, 0})
	expected = append(expected, []byte{0x80, 8})

	tests = append(tests, []byte{0, 0, 0, 0, 0, 0, 0, 8})
	expected = append(expected, []byte{0x01, 8})

	tests = append(tests, []byte{1, 1, 1, 0, 0, 0, 0, 0})
	expected = append(expected, []byte{0xe0, 0xf3, 1})

	tests = append(tests, []byte{0, 0, 1, 1, 1, 0, 0, 0})
	expected = append(expected, []byte{0x38, 0xf3, 1})

	tests = append(tests, []byte{0, 0, 1, 1, 0, 0, 0, 0})
	expected = append(expected, []byte{0x30, 1, 1})

	tests = append(tests, []byte{0, 1, 1, 1, 0, 1, 1, 1})
	expected = append(expected, []byte{0x77, 0xf3, 1, 0xf3, 1})

	tests = append(tests, []byte{1, 1, 1, 1, 1, 1, 1, 1})
	expected = append(expected, []byte{0xff, 0xf8, 1})

	tests = append(tests, []byte{1, 1, 1, 1, 1, 0, 1, 1})
	expected = append(expected, []byte{0xfb, 0xf5, 1, 1, 1})

	for i, _ := range tests {

		got, err := packLine(tests[i], false)

		if err != nil {

			t.Errorf("Unexpected error")
		}

		if slices.Compare(got, expected[i]) != 0 {

			t.Errorf("Unexpected return slice")
		}
	}
}
