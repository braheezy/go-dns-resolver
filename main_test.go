package main

import (
	"reflect"
	"testing"
)

func TestHeaderToBytes(t *testing.T) {
	testHeader := DNSHeader{
		id:          0x1314,
		flags:       0,
		questions:   1,
		answers:     0,
		authorities: 0,
		additional:  0,
	}
	actual := headerToBytes(&testHeader)
	expected := []byte{19, 20, 00, 00, 00, 01, 00, 00, 00, 00, 00, 00}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("headerToBytes actual = %v, expected = %v", actual, expected)
	}
}

func TestQuestionToBytes(t *testing.T) {
	testQuestion := DNSQuestion{
		name:  []byte("example.com"),
		type_: 1,
		class: 1,
	}
	actual := questionToBytes(&testQuestion)
	expected := []byte{
		101, 120, 97, 109, 112, 108, 101, 46, 99, 111, 109, 0, 1, 0, 1, 0, 0,
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("questionToBytes actual = %v, expected = %v", actual, expected)
	}
}

func TestEncodeDNSName(t *testing.T) {
	actual := encodeDNSName("google.com")
	expected := []byte("\x06google\x03com\x00")

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("encodeDNSName actual = %v, expected = %v", actual, expected)
	}

	if actual[0] != 6 {
		t.Errorf("encodeDNSName actual = %v, expected = %v", actual[0], 6)
	}
}
