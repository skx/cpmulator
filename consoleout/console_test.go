package consoleout

import "testing"

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
