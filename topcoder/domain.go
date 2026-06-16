package topcoder

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes TopCoder as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/topcoder-cli/topcoder"
func init() { kit.Register(Domain{}) }

// Domain is the TopCoder driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, hostnames, and identity for the binary.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:  "topcoder",
		Aliases: []string{"tc"},
		Hosts:   []string{Host, "www.topcoder.com", "api.topcoder.com"},
		Identity: kit.Identity{
			Binary: "tc",
			Short:  "Browse TopCoder challenges and member profiles",
			Long: `tc turns topcoder.com into a fast, scriptable command line.

Browse active challenges, look up challenge details, and check member profiles
from the public TopCoder REST API - no API key required.

Quick start:
  tc challenges                          list active challenges
  tc challenges --status completed -n 5  5 completed challenges
  tc challenge <uuid>                    one challenge by UUID
  tc user rng_58                         rng_58's public profile`,
			Site: Host,
			Repo: "https://github.com/tamnd/topcoder-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "challenges",
		Group:   "list",
		Summary: "List TopCoder challenges",
	}, listChallenges)

	kit.Handle(app, kit.OpMeta{
		Name:     "challenge",
		Group:    "read",
		Single:   true,
		Resolver: true,
		URIType:  "challenge",
		Summary:  "Fetch a challenge by UUID",
		Args:     []kit.Arg{{Name: "id", Help: "challenge UUID"}},
	}, getChallenge)

	kit.Handle(app, kit.OpMeta{
		Name:     "user",
		Group:    "read",
		Single:   true,
		Resolver: true,
		URIType:  "member",
		Summary:  "Fetch a member profile",
		Args:     []kit.Arg{{Name: "handle", Help: "member handle"}},
	}, getUser)
}

// newClient builds the Client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- input structs ---

type challengesInput struct {
	Status string  `kit:"flag" help:"filter by status: active|completed|draft|all" default:"active"`
	Type   string  `kit:"flag" help:"filter by type: COMPETITIVE_PROGRAMMING|DESIGN|DEVELOPMENT|QA"`
	Page   int     `kit:"flag" help:"page number (1-indexed)" default:"1"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"20"`
	Client *Client `kit:"inject"`
}

type challengeInput struct {
	ID     string  `kit:"arg" help:"challenge UUID"`
	Client *Client `kit:"inject"`
}

type userInput struct {
	Handle string  `kit:"arg" help:"member handle"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listChallenges(ctx context.Context, in challengesInput, emit func(Challenge) error) error {
	opts := ChallengesOptions{
		Status:  in.Status,
		Type:    in.Type,
		Page:    in.Page,
		PerPage: in.Limit,
	}
	// "all" status means no filter.
	if strings.ToLower(opts.Status) == "all" {
		opts.Status = ""
	}
	challenges, _, err := in.Client.ListChallenges(ctx, opts)
	if err != nil {
		return mapErr(err)
	}
	for _, ch := range challenges {
		if err := emit(ch); err != nil {
			return err
		}
	}
	return nil
}

func getChallenge(ctx context.Context, in challengeInput, emit func(*Challenge) error) error {
	ch, err := in.Client.GetChallenge(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(ch)
}

func getUser(ctx context.Context, in userInput, emit func(*Member) error) error {
	m, err := in.Client.GetMember(ctx, in.Handle)
	if err != nil {
		return mapErr(err)
	}
	return emit(m)
}

// --- Resolver ---

// uuidRE matches a v4 UUID.
var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Classify turns any accepted input into the canonical (uriType, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errs.Usage("topcoder: empty input")
	}
	// Full URL: https://www.topcoder.com/challenges/<uuid>
	if strings.Contains(input, "topcoder.com/challenges/") {
		parts := strings.Split(input, "/challenges/")
		if len(parts) == 2 {
			id := strings.Split(parts[1], "?")[0]
			if uuidRE.MatchString(id) {
				return "challenge", id, nil
			}
		}
	}
	// Full URL: https://www.topcoder.com/members/<handle>
	if strings.Contains(input, "topcoder.com/members/") {
		parts := strings.Split(input, "/members/")
		if len(parts) == 2 {
			return "member", strings.Split(parts[1], "?")[0], nil
		}
	}
	// UUID
	lower := strings.ToLower(input)
	if uuidRE.MatchString(lower) {
		return "challenge", lower, nil
	}
	// Legacy integer ID: treat as challenge
	if _, err := strconv.Atoi(input); err == nil {
		return "challenge", input, nil
	}
	// Handle-like
	if isHandle(input) {
		return "member", input, nil
	}
	return "", "", errs.Usage("topcoder: unrecognized reference: %q", input)
}

// Locate returns the canonical URL for a (uriType, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "challenge":
		return "https://www.topcoder.com/challenges/" + id, nil
	case "member":
		return "https://www.topcoder.com/members/" + id, nil
	default:
		return "", errs.Usage("topcoder has no resource type %q", uriType)
	}
}

// isHandle returns true when s looks like a valid TopCoder member handle:
// non-empty and composed of letters, digits, underscores, hyphens, dots.
func isHandle(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}

// mapErr converts library errors into kit error kinds with the correct exit codes.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return errs.NotFound("%s", err.Error())
	}
	if errors.Is(err, ErrRateLimited) || errors.Is(err, ErrBlocked) {
		return errs.RateLimited("%s", err.Error())
	}
	return err
}
