// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"

	"github.com/mattermost/mattermost-cloud/e2e/tests/state"
	"github.com/pkg/errors"
)

const (
	e2eTestGrafanaLogsURLTmpl     = `https://grafana.internal.mattermost.com/explore?orgId=1&left={"datasource":"PFB2D5CACEC34D62E","queries":[{"refId":"A","datasource":{"type":"loki","uid":"PFB2D5CACEC34D62E"},"editorMode":"code","expr":"{cluster=\"cnc\", namespace=\"e2e-mattermost-cloud-test\"} |= %60{{.ID}}%60","queryType":"range"}],"range":{"from":"now-24h","to":"now"}}`
	provisionerGrafanaLogsURLTmpl = `https://grafana.internal.mattermost.com/explore?orgId=1&left={"datasource":"PFB2D5CACEC34D62E","queries":[{"refId":"A","datasource":{"type":"loki","uid":"PFB2D5CACEC34D62E"},"editorMode":"code","expr":"{namespace=\"mattermost-cloud-test\", component=\"provisioner\"} |= %60{{.ID}}%60","queryType":"range"}],"range":{"from":"now-24h","to":"now"}}`
)

// getStringFromTemplate returns a string from a template and data provided.
func getStringFromTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", errors.Wrap(err, "error parsing template")
	}

	var result bytes.Buffer
	err = t.Execute(&result, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing template")
	}

	return result.String(), nil
}

type webhookPayloadAttachmentField struct {
	Short bool   `json:"short"`
	Title string `json:"title"`
	Value string `json:"value"`
}

type webhookPayloadAttachment struct {
	Title     string                          `json:"title"`
	TitleLink string                          `json:"title_link"`
	Color     string                          `json:"color"`
	Fields    []webhookPayloadAttachmentField `json:"fields"`
}

type webhookPayload struct {
	Username    string                     `json:"username"`
	IconURL     string                     `json:"icon_url"`
	IconEmoji   string                     `json:"icon_emoji"`
	Text        string                     `json:"text"`
	Attachments []webhookPayloadAttachment `json:"attachments"`
}

// sendWebhook sends a Mattermost webhook to the provided URL.
func sendWebhook(ctx context.Context, webhookURL string, payload *webhookPayload) error {
	if len(payload.Username) == 0 {
		return errors.New("payload username value not set")
	}
	if len(payload.Text) == 0 {
		return errors.New("payload text value not set")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal payload")
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send webhook")
	}

	return nil
}

func linkToGrafanaLogs(tmpl, id string) string {
	if len(id) == 0 {
		return "not set"
	}

	url, err := getStringFromTemplate(tmpl, struct {
		ID string
	}{
		ID: id,
	})
	if err != nil {
		return ""
	}
	return fmt.Sprintf("[`%s`](%s)", id, url)
}

// SendE2EResult sends the webhook with the provided icon and message. Errors on trying to send a
// message or if the webhook URL is not provided properly.
func SendE2EResult(ctx context.Context, icon, text, color string) error {
	webhookURL := os.Getenv("WEBHOOK_URL")
	_, err := url.ParseRequestURI(webhookURL)
	if err != nil {
		return errors.New("incorrect or empty webhook url")
	}

	payload := webhookPayload{
		Username:  "E2E",
		IconEmoji: icon,
		Text:      " ",
		Attachments: []webhookPayloadAttachment{{
			Title:     text,
			TitleLink: `https://grafana.internal.mattermost.com/goto/kWlEn-24k?orgId=1`,
			Color:     color,
			Fields: []webhookPayloadAttachmentField{
				{
					Short: true,
					Title: "TestID",
					Value: linkToGrafanaLogs(e2eTestGrafanaLogsURLTmpl, state.TestID),
				},
				{
					Short: true,
					Title: "ClusterID",
					Value: linkToGrafanaLogs(provisionerGrafanaLogsURLTmpl, state.ClusterID),
				},
				{
					Short: true,
					Title: "InstallationID",
					Value: linkToGrafanaLogs(provisionerGrafanaLogsURLTmpl, state.InstallationID),
				},
				{
					Short: true,
					Title: "Runtime",
					Value: (state.EndTime.Sub(state.StartTime)).String(),
				},
			},
		}},
	}

	if err := sendWebhook(ctx, webhookURL, &payload); err != nil {
		return errors.Wrap(err, "error sending notification webhook")
	}

	return nil
}
