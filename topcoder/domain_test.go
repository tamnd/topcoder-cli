package topcoder_test

import (
	"strings"
	"testing"

	"github.com/tamnd/topcoder-cli/topcoder"
)

func TestClassifyUUID(t *testing.T) {
	d := topcoder.Domain{}
	uriType, id, err := d.Classify("3e4c6634-a6d4-43ff-a8bc-5a05ac7cb7a5")
	if err != nil {
		t.Fatal(err)
	}
	if uriType != "challenge" {
		t.Errorf("type = %q, want challenge", uriType)
	}
	if id != "3e4c6634-a6d4-43ff-a8bc-5a05ac7cb7a5" {
		t.Errorf("id = %q", id)
	}
}

func TestClassifyHandle(t *testing.T) {
	d := topcoder.Domain{}
	uriType, id, err := d.Classify("rng_58")
	if err != nil {
		t.Fatal(err)
	}
	if uriType != "member" {
		t.Errorf("type = %q, want member", uriType)
	}
	if id != "rng_58" {
		t.Errorf("id = %q, want rng_58", id)
	}
}

func TestClassifyChallengeURL(t *testing.T) {
	d := topcoder.Domain{}
	uriType, id, err := d.Classify("https://www.topcoder.com/challenges/3e4c6634-a6d4-43ff-a8bc-5a05ac7cb7a5")
	if err != nil {
		t.Fatal(err)
	}
	if uriType != "challenge" {
		t.Errorf("type = %q, want challenge", uriType)
	}
	if id != "3e4c6634-a6d4-43ff-a8bc-5a05ac7cb7a5" {
		t.Errorf("id = %q", id)
	}
}

func TestClassifyMemberURL(t *testing.T) {
	d := topcoder.Domain{}
	uriType, id, err := d.Classify("https://www.topcoder.com/members/tourist")
	if err != nil {
		t.Fatal(err)
	}
	if uriType != "member" {
		t.Errorf("type = %q, want member", uriType)
	}
	if id != "tourist" {
		t.Errorf("id = %q, want tourist", id)
	}
}

func TestClassifyLegacyID(t *testing.T) {
	d := topcoder.Domain{}
	uriType, id, err := d.Classify("30123456")
	if err != nil {
		t.Fatal(err)
	}
	if uriType != "challenge" {
		t.Errorf("type = %q, want challenge", uriType)
	}
	if id != "30123456" {
		t.Errorf("id = %q, want 30123456", id)
	}
}

func TestClassifyEmptyInput(t *testing.T) {
	d := topcoder.Domain{}
	_, _, err := d.Classify("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestLocateChallenge(t *testing.T) {
	d := topcoder.Domain{}
	u, err := d.Locate("challenge", "abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "/challenges/abc-123") {
		t.Errorf("URL = %q, want /challenges/abc-123", u)
	}
}

func TestLocateMember(t *testing.T) {
	d := topcoder.Domain{}
	u, err := d.Locate("member", "tourist")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "/members/tourist") {
		t.Errorf("URL = %q, want /members/tourist", u)
	}
}

func TestLocateUnknownType(t *testing.T) {
	d := topcoder.Domain{}
	_, err := d.Locate("unknown", "x")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestDomainInfoScheme(t *testing.T) {
	d := topcoder.Domain{}
	info := d.Info()
	if info.Scheme != "topcoder" {
		t.Errorf("Scheme = %q, want topcoder", info.Scheme)
	}
	if info.Identity.Binary != "tc" {
		t.Errorf("Binary = %q, want tc", info.Identity.Binary)
	}
}
