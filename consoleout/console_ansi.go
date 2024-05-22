package consoleout

import "fmt"

// AnsiOutputDriver holds our state.
type AnsiOutputDriver struct {
}

func (ad AnsiOutputDriver) GetName() string {
	return "ansi"
}
func (ad AnsiOutputDriver) PutCharacter(c byte) {
	fmt.Printf("%c", c)
}
func init() {
	Register("ansi", func() ConsoleDriver {
		return &AnsiOutputDriver{}
	})
}
