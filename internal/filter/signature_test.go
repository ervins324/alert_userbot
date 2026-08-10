package filter

import "testing"

const channelSignature = "Підписатись 👉 🚀ППО | РАДАР (https://t.me/mon1tor_ua)\n@mon1tor_ua"

func TestRemoveSignatureFromPost(t *testing.T) {
	post := "🔴 Ракетна небезпека в Києві!\n\n" + channelSignature

	cleaned, removed := RemoveSignature(post)
	if !removed {
		t.Fatal("expected signature to be removed")
	}
	if cleaned != "🔴 Ракетна небезпека в Києві!" {
		t.Errorf("unexpected cleaned text: %q", cleaned)
	}
}

func TestRemoveSignatureWithoutHandle(t *testing.T) {
	post := "❗️❗❗Загроза пуску балістичних ракет \"Іскандер-М\"/\"С-300\" з Курської області.\nПідписатись 👉 🚀ППО | РАДАР"

	cleaned, removed := RemoveSignature(post)
	if !removed {
		t.Fatal("expected signature to be removed")
	}
	want := "❗️❗❗Загроза пуску балістичних ракет \"Іскандер-М\"/\"С-300\" з Курської області."
	if cleaned != want {
		t.Errorf("unexpected cleaned text: %q", cleaned)
	}
}

func TestRemoveSignatureOnlyFooter(t *testing.T) {
	cleaned, removed := RemoveSignature(channelSignature)
	if !removed {
		t.Fatal("expected signature to be removed")
	}
	if cleaned != "" {
		t.Errorf("expected empty text, got %q", cleaned)
	}
}

func TestRemoveSignaturePlainMessage(t *testing.T) {
	text := "Просто новина без підпису"
	cleaned, removed := RemoveSignature(text)
	if removed {
		t.Error("signature should not be found")
	}
	if cleaned != text {
		t.Errorf("text should be unchanged, got %q", cleaned)
	}
}

func TestHasSignature(t *testing.T) {
	if !HasSignature("Пост з підписом\n" + channelSignature) {
		t.Error("expected signature detected")
	}
	if HasSignature("Пост без підпису") {
		t.Error("expected no signature")
	}
}
