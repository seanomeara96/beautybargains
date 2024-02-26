package models

type Profile struct {
	Photo    string
	Username string
}

var DummyProfile = Profile{"https://vogue.implicitdev.com/images/liana-agron69e2c.png", "Leanne"}

type ExtraImage struct {
	Src string
	Alt string
}

var DummyExtraImages = []ExtraImage{
	ExtraImage{"https://vogue.implicitdev.com/images/liana-agron69e2c.png", "dummy image"},
	ExtraImage{"https://vogue.implicitdev.com/images/liana-agron69e2c.png", "dummy image"},
}

type Content struct {
	Summary     string
	TimeElapsed string
	ExtraImages *[]ExtraImage // optional
	ExtraText   *string       // optional
}

var DummyExtraText = "Ours is a life of constant reruns. We're always circling back to where we'd we started, then starting all over again. Even if we don't run extra laps that day, we surely will come back for more of the same another day soon."

var DummyContent = Content{
	"added 2 new photos",
	"4 Days Ago",
	&DummyExtraImages,
	&DummyExtraText,
}

type EventMeta struct {
	Src   *string
	Likes int
}

var DummyEventMetaSrc = "/"

var DummyEventMeta = EventMeta{&DummyEventMetaSrc, 0}

type Event struct {
	Profile Profile
	Content Content
	Meta    EventMeta
}

var DummyEvent = Event{DummyProfile, DummyContent, DummyEventMeta}

type FeedPage struct {
	Events []Event
}
