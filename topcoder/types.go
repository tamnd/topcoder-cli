package topcoder

import (
	"errors"
	"sort"
	"time"
)

// ErrNotFound is returned when the API responds with HTTP 404.
var ErrNotFound = errors.New("not found")

// ErrRateLimited is returned after exhausting retries on HTTP 429.
var ErrRateLimited = errors.New("rate limited (HTTP 429)")

// ErrBlocked is returned when the API returns HTTP 503 (CDN/WAF block).
var ErrBlocked = errors.New("blocked (HTTP 503)")

// Challenge is one TopCoder challenge record.
type Challenge struct {
	ID          string    `json:"id"`
	LegacyID    int       `json:"legacyId,omitempty"`
	Name        string    `json:"name"`
	Type        string    `json:"type,omitempty"`
	Track       string    `json:"track,omitempty"`
	Status      string    `json:"status"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Tags        []string  `json:"tags,omitempty"`
	Prize1      int       `json:"prize1,omitempty"`
	Prize2      int       `json:"prize2,omitempty"`
	Deadline    string    `json:"deadline,omitempty"`
	Registrants int       `json:"registrants,omitempty"`
	Submissions int       `json:"submissions,omitempty"`
	ReviewType  string    `json:"reviewType,omitempty"`
	Description string    `json:"description,omitempty"`
}

// Member is one TopCoder member public profile.
type Member struct {
	Handle      string    `json:"handle"`
	UserID      int64     `json:"userId,omitempty"`
	Country     string    `json:"country,omitempty"`
	Description string    `json:"description,omitempty"`
	PhotoURL    string    `json:"photoUrl,omitempty"`
	Tracks      []string  `json:"tracks,omitempty"`
	Wins        int       `json:"wins,omitempty"`
	MaxRating   int       `json:"maxRating,omitempty"`
	RatingTrack string    `json:"ratingTrack,omitempty"`
	Skills      []string  `json:"skills,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// --- raw API shapes ---

type rawChallenge struct {
	ID          string     `json:"id"`
	LegacyID    int        `json:"legacyId"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Track       string     `json:"track"`
	Status      string     `json:"status"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
	Tags        []string   `json:"tags"`
	PrizeSets   []prizeSet `json:"prizeSets"`
	Phases      []phase    `json:"phases"`
	NumReg      int        `json:"numOfRegistrants"`
	NumSub      int        `json:"numOfSubmissions"`
	ReviewType  string     `json:"reviewType"`
	Description string     `json:"description"`
}

type prizeSet struct {
	Type   string  `json:"type"`
	Prizes []prize `json:"prizes"`
}

type prize struct {
	Value int    `json:"value"`
	Type  string `json:"type"`
}

type phase struct {
	Name             string    `json:"name"`
	ScheduledEndDate time.Time `json:"scheduledEndDate"`
	IsOpen           bool      `json:"isOpen"`
}

type rawMember struct {
	Handle      string     `json:"handle"`
	UserID      int64      `json:"userId"`
	Country     string     `json:"country"`
	Description string     `json:"description"`
	PhotoURL    string     `json:"photoURL"`
	Tracks      []string   `json:"tracks"`
	Wins        int        `json:"wins"`
	MaxRating   *maxRating `json:"maxRating"`
	Skills      []skill    `json:"skills"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type maxRating struct {
	Rating   int    `json:"rating"`
	Track    string `json:"track"`
	SubTrack string `json:"subTrack"`
}

type skill struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// fromRawChallenge converts the wire format to the public Challenge type.
func fromRawChallenge(r rawChallenge) Challenge {
	c := Challenge{
		ID:          r.ID,
		LegacyID:    r.LegacyID,
		Name:        r.Name,
		Type:        r.Type,
		Track:       r.Track,
		Status:      r.Status,
		Created:     r.Created,
		Updated:     r.Updated,
		Tags:        r.Tags,
		Registrants: r.NumReg,
		Submissions: r.NumSub,
		ReviewType:  r.ReviewType,
		Description: r.Description,
	}

	// Extract prize1 and prize2 from the placement prizeSet.
	for _, ps := range r.PrizeSets {
		if ps.Type == "placement" || len(r.PrizeSets) == 1 {
			if len(ps.Prizes) > 0 {
				c.Prize1 = ps.Prizes[0].Value
			}
			if len(ps.Prizes) > 1 {
				c.Prize2 = ps.Prizes[1].Value
			}
			break
		}
	}
	if c.Prize1 == 0 && len(r.PrizeSets) > 0 && len(r.PrizeSets[0].Prizes) > 0 {
		c.Prize1 = r.PrizeSets[0].Prizes[0].Value
	}

	// Earliest open phase end date becomes the deadline.
	var earliest time.Time
	for _, p := range r.Phases {
		if p.IsOpen && !p.ScheduledEndDate.IsZero() {
			if earliest.IsZero() || p.ScheduledEndDate.Before(earliest) {
				earliest = p.ScheduledEndDate
			}
		}
	}
	if !earliest.IsZero() {
		c.Deadline = earliest.Format("2006-01-02")
	}

	return c
}

// fromRawMember converts the wire format to the public Member type.
func fromRawMember(r rawMember) Member {
	m := Member{
		Handle:      r.Handle,
		UserID:      r.UserID,
		Country:     r.Country,
		Description: r.Description,
		PhotoURL:    r.PhotoURL,
		Tracks:      r.Tracks,
		Wins:        r.Wins,
		CreatedAt:   r.CreatedAt,
	}
	if r.MaxRating != nil {
		m.MaxRating = r.MaxRating.Rating
		m.RatingTrack = r.MaxRating.Track
	}

	// Top 5 skills sorted by score descending.
	sk := make([]skill, len(r.Skills))
	copy(sk, r.Skills)
	sort.Slice(sk, func(i, j int) bool { return sk[i].Score > sk[j].Score })
	max := 5
	if len(sk) < max {
		max = len(sk)
	}
	for _, s := range sk[:max] {
		if s.Name != "" {
			m.Skills = append(m.Skills, s.Name)
		}
	}

	return m
}
