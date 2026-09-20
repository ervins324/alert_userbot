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

func TestRemoveSignatureUserKyivMonitorCase(t *testing.T) {
	post := "З Вишневого на Петропавлівську Борщагівку.\n\n@kyiv_monitor1"

	cleaned, removed := RemoveSignature(post)
	if !removed {
		t.Fatal("expected signature @kyiv_monitor1 to be removed")
	}
	want := "З Вишневого на Петропавлівську Борщагівку."
	if cleaned != want {
		t.Errorf("expected %q, got %q", want, cleaned)
	}
}

func TestRemoveSignatureTrailingLink(t *testing.T) {
	post := "Збито ворожий дрон над областю.\n\nt.me/kyiv_monitor1"

	cleaned, removed := RemoveSignature(post)
	if !removed {
		t.Fatal("expected t.me link to be removed")
	}
	want := "Збито ворожий дрон над областю."
	if cleaned != want {
		t.Errorf("expected %q, got %q", want, cleaned)
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

	cleaned2, removed2 := RemoveSignature("@kyiv_monitor1")
	if !removed2 {
		t.Fatal("expected handle to be removed")
	}
	if cleaned2 != "" {
		t.Errorf("expected empty text, got %q", cleaned2)
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

func TestRemoveSignatureHandleInsideSentence(t *testing.T) {
	text := "За інформацією @kyiv_monitor1 в центрі міста працює ППО."
	cleaned, removed := RemoveSignature(text)
	if removed {
		t.Error("inline handle should not be removed as signature")
	}
	if cleaned != text {
		t.Errorf("text should be unchanged, got %q", cleaned)
	}
}

func TestHasSignature(t *testing.T) {
	if !HasSignature("Пост з підписом\n" + channelSignature) {
		t.Error("expected signature detected")
	}
	if !HasSignature("З Вишневого на Петропавлівську Борщагівку.\n\n@kyiv_monitor1") {
		t.Error("expected @kyiv_monitor1 detected")
	}
	if HasSignature("Просто новина без підпису") {
		t.Error("expected no signature")
	}
}
