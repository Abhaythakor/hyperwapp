package model_test

import (
	"github.com/Abhaythakor/hyperwapp/model"
	"testing"
)

func TestOfflineInputSchema(t *testing.T) {
	tests := []struct {
		name     string
		input    model.OfflineInput
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name: "Valid input",
			input: model.OfflineInput{
				Domain:  "example.com",
				URL:     "https://example.com/path",
				Headers: map[string][]string{"Content-Type": {"text/html"}},
				Body:    []byte("<html>"),
			},
			wantErr: false,
		},
		{
			name: "Missing Domain",
			input: model.OfflineInput{
				URL:     "https://example.com/path",
				Headers: map[string][]string{"Content-Type": {"text/html"}},
				Body:    []byte("<html>"),
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "OfflineInput: Domain cannot be empty"
			},
		},
		{
			name: "Empty Headers",
			input: model.OfflineInput{
				Domain:  "example.com",
				URL:     "https://example.com/path",
				Headers: map[string][]string{}, // Explicitly empty map
				Body:    []byte("<html>"),
			},
			wantErr: false, // Empty headers are allowed, but the map must exist
		},
		{
			name: "Nil Headers",
			input: model.OfflineInput{
				Domain:  "example.com",
				URL:     "https://example.com/path",
				Headers: nil, // Explicitly nil headers
				Body:    []byte("<html>"),
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "OfflineInput: Headers map must not be nil"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate() // Call the method directly
			if (err != nil) != tt.wantErr {
				t.Fatalf("tt.input.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("tt.input.Validate() error = %v, did not pass errCheck", err)
			}
		})
	}
}
