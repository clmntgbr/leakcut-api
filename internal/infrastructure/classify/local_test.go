package classify

import (
	"context"
	"testing"

	"go-api/internal/domain/port"
)

func TestLocalDetector_Classify(t *testing.T) {
	detector := NewLocalDetector()
	cases := []struct {
		text     string
		status   string
		hit      bool
		category string
	}{
		{text: "", status: "skipped"},
		{text: "hello world", status: "success"},
		{text: "contact me at ada@example.com please", status: "success", hit: true, category: "email"},
		{text: "token sk-abcdefghijklmnopqrstuvwxyz012345", status: "success", hit: true, category: "api_key"},
		{text: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.signaturehere", status: "success", hit: true, category: "jwt"},
		{text: "postgres://user:secret@localhost:5432/app", status: "success", hit: true, category: "connection_string"},
		{text: "06 12 34 56 78", status: "success", hit: true, category: "phone"},
		{text: "+33 6 12 34 56 78", status: "success", hit: true, category: "phone"},
		{text: "+1 202-555-0147", status: "success", hit: true, category: "phone"},
		{text: "48.8566", status: "success"},
		{text: "48.856614, 2.352221", status: "success"},
		{text: "lat: 48.8566 lon: 2.3522", status: "success"},
		{text: "48°51'24\"N 2°21'03\"E", status: "success"},
		{text: "N 45.5017 W 73.5673", status: "success"},
		{text: "ES_PASSWORD}@${POSTGRES_NAME}:543 sidecar up TMQ_DEFAULT_PASS}@rabbitmq:5672/", status: "success", hit: true, category: "password"},
		{text: "amqp://${RABBITMQ_DEFAULT_USER}:${RABBITMQ_DEFAULT_PASS}@rabbitmq:5672/", status: "success", hit: true, category: "connection_string"},
		{text: "CLERK_SECRET_KEY=sk_test_abcdefghijklmnopqrstuv", status: "success", hit: true, category: "api_key"},
		{text: "CLERK_WEBHOOK_SECRET=whsec_abcdefghijklmnopqrstuv", status: "success", hit: true, category: "webhook_secret"},
		{text: "APP_DB_PASSWORD=hunter2", status: "success", hit: true, category: "password"},
		{text: "service:s3cret@db.internal:6432", status: "success", hit: true, category: "connection_string"},
		{text: "${CACHE_HOST}:11211", status: "success", hit: true, category: "connection_string"},
		{text: "CORS_ALLOWED_ORIGINS=http://localhost:3000", status: "success"},
	}

	for _, tc := range cases {
		out, err := detector.Classify(context.Background(), []port.ClassifyFrame{{
			FrameID: "frame-1",
			Text:    tc.text,
		}}, 0.7)
		if err != nil {
			t.Fatalf("classify %q: %v", tc.text, err)
		}
		if out[0].Status != tc.status {
			t.Fatalf("status for %q: got %s want %s", tc.text, out[0].Status, tc.status)
		}
		if out[0].Confidential != tc.hit {
			t.Fatalf("confidential for %q: got %t want %t", tc.text, out[0].Confidential, tc.hit)
		}
		if tc.category == "" {
			continue
		}
		found := false
		for _, category := range out[0].Categories {
			if category.Name == tc.category {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing category %s for %q", tc.category, tc.text)
		}
	}
}
