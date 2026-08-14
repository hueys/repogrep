package cli

import "testing"

func TestValidateUnstarFlag(t *testing.T) {
	cases := []struct {
		unstar, force bool
		wantErr       bool
	}{
		{unstar: false, force: false, wantErr: false},
		{unstar: false, force: true, wantErr: false},
		{unstar: true, force: true, wantErr: false},
		{unstar: true, force: false, wantErr: true},
	}
	for _, c := range cases {
		err := validateUnstarFlag(c.unstar, c.force)
		if (err != nil) != c.wantErr {
			t.Errorf("validateUnstarFlag(%v, %v) = %v, wantErr=%v", c.unstar, c.force, err, c.wantErr)
		}
	}
}
