package scraper

import (
	"os"
	"testing"
)

func TestParseRealChannelSample(t *testing.T) {
	body, err := os.ReadFile("testdata/mon1tor_sample.html")
	if err != nil {
		t.Skipf("no sample fixture: %v", err)
	}

	msgs, err := parseMessages(body)
	if err != nil {
		t.Fatalf("parseMessages error: %v", err)
	}

	if len(msgs) == 0 {
		t.Fatal("expected to parse at least one message")
	}

	for _, m := range msgs {
		if m.ID == 0 {
			t.Error("message ID is 0")
		}
		if m.Text == "" && m.PhotoURL == "" {
			t.Errorf("message %d has neither text nor photo", m.ID)
		}
	}

	// IDs must be strictly increasing in page order.
	for i := 1; i < len(msgs); i++ {
		if msgs[i].ID <= msgs[i-1].ID {
			t.Errorf("message IDs not increasing: %d then %d", msgs[i-1].ID, msgs[i].ID)
		}
	}
}

func TestParsePhotoMessage(t *testing.T) {
	body := []byte(`
		<div class="tgme_widget_message_wrap js-widget_message_wrap">
			<div class="tgme_widget_message text_not_supported_wrap js-widget_message" data-post="mon1tor_ua/70000">
				<div class="tgme_widget_message_photo_wrap animated_image_wrap">
					<a class="tgme_widget_message_photo_link" href="https://cdn4.telesco.pe/file/abc.jpg">
						<img src="https://cdn4.telesco.pe/file/abc.jpg"/>
					</a>
				</div>
				<div class="tgme_widget_message_text js-message_text">Мапа тривог Київ</div>
			</div>
		</div>
		<div class="tgme_widget_message_wrap js-widget_message_wrap">
			<div class="tgme_widget_message js-widget_message" data-post="mon1tor_ua/70001">
				<div class="tgme_widget_message_text js-message_text">Текст без фото</div>
			</div>
		</div>
	`)

	msgs, err := parseMessages(body)
	if err != nil {
		t.Fatalf("parseMessages error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	if msgs[0].ID != 70000 || msgs[0].PhotoURL != "https://cdn4.telesco.pe/file/abc.jpg" {
		t.Errorf("photo message wrong: %+v", msgs[0])
	}
	if msgs[0].Text != "Мапа тривог Київ" {
		t.Errorf("photo caption wrong: %q", msgs[0].Text)
	}
	if msgs[1].ID != 70001 || msgs[1].PhotoURL != "" || msgs[1].Text != "Текст без фото" {
		t.Errorf("text message wrong: %+v", msgs[1])
	}
}

func TestPostID(t *testing.T) {
	if got := postID("mon1tor_ua/69801"); got != 69801 {
		t.Errorf("postID = %d", got)
	}
	if got := postID("no-slash"); got != 0 {
		t.Errorf("postID invalid = %d", got)
	}
}
