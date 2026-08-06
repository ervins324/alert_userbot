package forwarder

import (
	"testing"
)

func TestTextFilterShouldSkip(t *testing.T) {
	filter := NewTextFilter(nil)

	cases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "donation post",
			text:     "Терміновий збір на 7 FPV дронів. 🔗 https://send.monobank.ua/jar/8bPUwpYCMK",
			expected: true,
		},
		{
			name:     "donation post (donaт)",
			text:     "Перед тим як лягати відпочивати нам дуже потрібен ваш донат для ГУР МОУ на FPV",
			expected: true,
		},
		{
			name:     "bank jar card",
			text:     "💳Номер картки банки 4874 1000 3147 3609",
			expected: true,
		},
		{
			name:     "monobank link",
			text:     "Посилання на банку: https://send.monobank.ua/jar/8bPUwpYCMK",
			expected: true,
		},
		{
			name:     "normal alert info",
			text:     "🔴Ракета Х-31П на Одесу/Чорноморськ. Загроза пуску балістичних ракет.",
			expected: false,
		},
		{
			name:     "air alert message",
			text:     "Повітряна тривога в Києві. Переходьте в укриття.",
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filter.ShouldSkip(tc.text); got != tc.expected {
				t.Errorf("ShouldSkip() = %v; want %v", got, tc.expected)
			}
		})
	}
}

func TestTextFilterCustomPatterns(t *testing.T) {
	filter := NewTextFilter([]string{"собіраем", "кава"})

	if !filter.ShouldSkip("підтримаєте на каву") {
		t.Error("expected custom pattern 'кава' to match")
	}
	if filter.ShouldSkip("просто новина про погоду") {
		t.Error("unrelated message should not match")
	}
}
