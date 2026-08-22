//go:build race

package json_test

// The race detector turns on checkptr, which makes the backend's unsafe pointer
// arithmetic fatal rather than merely out of bounds. See raceUnsafeCases.
const raceDetectorEnabled = true
