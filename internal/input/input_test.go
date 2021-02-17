package input

import (
	"errors"
	"testing"
)

func CheckError(out error, want error, t *testing.T) {
	t.Helper()
	if out == nil && want == nil {
		return
	}
	if out != nil && want == nil {
		t.Errorf("got %v, want %v", out.Error(), "")
	}
	if out == nil && want != nil {
		t.Errorf("got %v, want %v", "", want.Error())
	}
	if out != nil && want != nil && out.Error() != want.Error() {
		t.Errorf("got %v, want %v", out.Error(), want.Error())
	}

}
func TestExtractInputInitialByteCount(T *testing.T) {
	data := []struct {
		name            string
		inSize          string
		inTotal         int
		inErrorIfHigher bool
		err             error
	}{
		{
			name:            "[Donwloaded  - error if higher]100kb input with 200kb limit shouldn't return error test",
			inSize:          "100kb",
			inTotal:         204800,
			inErrorIfHigher: true,
		},
		{
			name:            "[Donwloaded -  error if higher]300kb input with 200kb limit should return error test",
			inSize:          "300kb",
			inTotal:         204800,
			inErrorIfHigher: true,
			err:             errors.New("initial downloaded can not be higher than the torrent size"),
		},
		{
			name:            "[Uploaded]100kb input with 200kb limit shouldn't return error test",
			inSize:          "100kb",
			inTotal:         204800,
			inErrorIfHigher: false,
		},
		{
			name:            "[Uploaded]300kb input with 200kb limit shouldn't return error test",
			inSize:          "300kb",
			inTotal:         204800,
			inErrorIfHigher: false,
		},
		{
			name:            "[Donwloaded] -100kb should return negative number error test",
			inSize:          "-100kb",
			inTotal:         204800,
			inErrorIfHigher: true,
			err:             errors.New("initial value can not be negative"),
		},
		{
			name:            "[Uploaded] -100kb should return negative number error test",
			inSize:          "-100kb",
			inTotal:         204800,
			inErrorIfHigher: false,
			err:             errors.New("initial value can not be negative"),
		},
	}

	for _, td := range data {
		T.Run(td.name, func(t *testing.T) {
			_, err := extractInputInitialByteCount(td.inSize, td.inTotal, td.inErrorIfHigher)
			CheckError(err, td.err, t)
		})
	}
}
func TestStrSize2ByteSize(T *testing.T) {

	data := []struct {
		name        string
		in          string
		inTotalSize int
		out         int
		err         error
	}{
		{
			name:        "100kb test",
			in:          "100kb",
			inTotalSize: 100,
			out:         102400,
		},
		{
			name:        "1kb test",
			in:          "1kb",
			inTotalSize: 0,
			out:         1024,
		},
		{
			name:        "1mb test",
			in:          "1mb",
			inTotalSize: 0,
			out:         1048576,
		},
		{
			name:        "1gb test",
			in:          "1gb",
			inTotalSize: 0,
			out:         1073741824,
		},
		{
			name:        "1.5gb test",
			in:          "1.5gb",
			inTotalSize: 0,
			out:         1610612736,
		},
		{
			name:        "1tb test",
			in:          "1tb",
			inTotalSize: 0,
			out:         1099511627776,
		},
		{
			name:        "1b test",
			in:          "1b",
			inTotalSize: 0,
			out:         1,
		},
		{
			name:        "10xb test",
			in:          "10xb",
			inTotalSize: 0,
			err:         errors.New("invalid input size"),
		},
		{
			name:        `100% test`,
			in:          "100%",
			inTotalSize: 10737418240,
			out:         10737418240,
		},
		{
			name:        `55% test`,
			in:          "55%",
			inTotalSize: 943718400,
			out:         519045120,
		},
		{
			name: `5kg test`,
			in:   "5kg",
			err:  errors.New("Size not found"),
		},
		{
			name: `-1% test`,
			in:   "-1%",
			err:  errors.New("percent value must be in (0-100)"),
		},
		{
			name: `101% test`,
			in:   "101%",
			err:  errors.New("percent value must be in (0-100)"),
		},
		{
			name: `a% test`,
			in:   "a%",
			err:  errors.New("percent value must be in (0-100)"),
		},
	}

	for _, td := range data {
		T.Run(td.name, func(t *testing.T) {
			got, err := strSize2ByteSize(td.in, td.inTotalSize)
			if td.err != nil {
				if td.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), td.err.Error())
				}
			}
			if got != td.out {
				t.Errorf("got %v, want %v", got, td.out)
			}
		})
	}
}

func TestExtractInputByteSpeed(T *testing.T) {

	data := []struct {
		name     string
		speed    string
		expected int
		err      error
	}{
		{
			name:     "1kbps test",
			speed:    "1kbps",
			expected: 1024,
		},
		{
			name:     "1024kbps test",
			speed:    "1024kbps",
			expected: 1048576,
		},
		{
			name:     "1mbps test",
			speed:    "1mbps",
			expected: 1048576,
		},
		{
			name:     "2.5mbps test",
			speed:    "2.5mbps",
			expected: 2621440,
		},
		{
			name:  "2.5tbps test",
			speed: "2.5tbps",
			err:   errors.New("speed must be in [kbps mbps]"),
		},
		{
			name:  "-akbps test",
			speed: "-akbps",
			err:   errors.New("invalid speed number"),
		},
		{
			name:  "-10kbps test",
			speed: "-10kbps",
			err:   errors.New("speed can not be negative"),
		},
	}

	for _, td := range data {
		T.Run(td.name, func(t *testing.T) {
			got, err := extractInputByteSpeed(td.speed)
			if td.err != nil {
				if td.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), td.err.Error())
				}
			}

			if got != td.expected {
				t.Errorf("got %v, want %v", got, td.expected)
			}
		})
	}
}
