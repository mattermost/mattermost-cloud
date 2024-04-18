package provisioner

import (
	"testing"
)

func TestModifyAMISuffix(t *testing.T) {
	testCases := []struct {
		name      string
		ami       string
		archLabel string
		wantAMI   string
	}{
		{
			name:      "AMI starts with ami-",
			ami:       "ami-12345",
			archLabel: "arm64",
			wantAMI:   "ami-12345",
		},
		{
			name:      "No suffix, no label",
			ami:       "custom-ubuntu",
			archLabel: "",
			wantAMI:   "custom-ubuntu-amd64",
		},
		{
			name:      "No suffix, with arm64 label",
			ami:       "custom-ubuntu",
			archLabel: "arm64",
			wantAMI:   "custom-ubuntu-arm64",
		},
		{
			name:      "With amd64 suffix, no label change",
			ami:       "custom-ubuntu-amd64",
			archLabel: "amd64",
			wantAMI:   "custom-ubuntu-amd64",
		},
		{
			name:      "With amd64 suffix, change to arm64",
			ami:       "custom-ubuntu-amd64",
			archLabel: "arm64",
			wantAMI:   "custom-ubuntu-arm64",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotAMI := ModifyAMISuffix(tc.ami, tc.archLabel)
			if gotAMI != tc.wantAMI {
				t.Errorf("ModifyAMISuffix(%q, %q) = %q, want %q", tc.ami, tc.archLabel, gotAMI, tc.wantAMI)
			}
		})
	}
}
