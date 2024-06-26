package consoleout

import (
	"bytes"
	"testing"
)

// TestName ensures we can lookup a driver by name
func TestName(t *testing.T) {

	valid := []string{"ansi", "adm-3a"}

	for _, nm := range valid {

		d, e := New(nm)
		if e != nil {
			t.Fatalf("failed to lookup driver by name %s:%s", nm, e)
		}
		if d.GetName() != nm {
			t.Fatalf("%s != %s", d.GetName(), nm)
		}
	}

	// Lookup a driver that wont exist
	_, err := New("foo.bar.ba")
	if err == nil {
		t.Fatalf("we got a driver that shouldn't exist")
	}
}

// TestChangeDriver ensures we can change a driver
func TestChangeDriver(t *testing.T) {

	// Start with a known-good driver
	ansi, err := New("ansi")
	if err != nil {
		t.Fatalf("failed to load starting driver %s", err)
	}

	// Change to another known-good driver
	err = ansi.ChangeDriver("adm-3a")
	if err != nil {
		t.Fatalf("failed to change to new driver %s", err)
	}
	if ansi.GetName() != "adm-3a" {
		t.Fatalf("driver change didnt work?")
	}

	// Change to a bogus driver
	err = ansi.ChangeDriver("fofdsf-fsdfsd-fsdfdsf-")
	if err == nil {
		t.Fatalf("expected failure to change to new driver, didn't happen")
	}
	if ansi.GetName() != "adm-3a" {
		t.Fatalf("driver changed unexpectedly")
	}
}

// TestOutput ensures that our two "real" drivers output, as expected
func TestOutput(t *testing.T) {

	// Drivers that should produce output
	valid := []string{"ansi", "adm-3a"}

	for _, nm := range valid {

		d, e := New(nm)
		if e != nil {
			t.Fatalf("failed to lookup driver by name %s:%s", nm, e)
		}

		// ensure we redirect the output
		tmp := &bytes.Buffer{}

		d.driver.SetWriter(tmp)

		for _, c := range "Steve Kemp" {
			d.PutCharacter(byte(c))
		}

		// Test we got the output we expected
		if tmp.String() != "Steve Kemp" {
			t.Fatalf("output driver %s produced '%s'", d.GetName(), tmp.String())
		}
	}

}

// TestNull ensures nothing is written by the null output driver
func TestNull(t *testing.T) {

	// Start with a known-good driver
	null, err := New("null")
	if err != nil {
		t.Fatalf("failed to load starting driver %s", err)
	}
	if null.GetName() != "null" {
		t.Fatalf("null driver has the wrong name")
	}

	// ensure we redirect the output
	tmp := &bytes.Buffer{}

	null.driver.SetWriter(tmp)

	null.PutCharacter('s')

	if tmp.String() != "" {
		t.Fatalf("got output, expected none: '%s'", tmp.String())
	}
}

func TestList(t *testing.T) {
	x, _ := New("foo")

	valid := x.GetDrivers()

	if len(valid) != 2 {
		t.Fatalf("unexpected number of console drivers")
	}
}
