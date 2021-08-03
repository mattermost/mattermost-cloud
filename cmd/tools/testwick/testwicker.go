// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	cmodel "github.com/mattermost/mattermost-cloud/model"
	mmodel "github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type TestWicker struct {
	provisionerClient *cmodel.Client
	mmClient          *mmodel.Client4
	logger            *logrus.Logger
	installation      *cmodel.InstallationDTO
	userID            string
	channelID         string
	teamID            string
	Error             error
}

func NewTestWicker(provisionerClient *cmodel.Client, logger *logrus.Logger) *TestWicker {
	return &TestWicker{
		logger:            logger,
		provisionerClient: provisionerClient,
	}
}

// CreateInstallation interacts with provisioner and uses the provided request
// to create a new installation
func (w *TestWicker) CreateInstallation(request *cmodel.CreateInstallationRequest) *TestWicker {
	w.logger.WithField("DNS", request.DNS).Info("Creating installation")
	if err := request.Validate(); err != nil {
		w.Error = errors.Wrap(err, "CreateInstallationRequest.Validate")
		return w
	}
	i, err := w.provisionerClient.CreateInstallation(request)
	if err != nil {
		w.Error = errors.Wrap(err, "client.CreateInstallation")
		return w
	}
	w.installation = i
	return w
}

// DeleteInstallation interacts with provisioner and uses the provided request
// to delete an installation
func (w *TestWicker) DeleteInstallation() *TestWicker {
	w.logger.WithField("ID", w.installation.ID).Info("Deleting installation")
	err := w.provisionerClient.DeleteInstallation(w.installation.ID)
	if err != nil {
		w.Error = errors.Wrap(err, "client.DeleteInstallation")
		return w
	}
	return w
}

// PostMessage post messages to the different channels which are created
// automatically by the wicker
func (w *TestWicker) PostMessage(samples int) *TestWicker {
	w.logger.WithField("DNS", w.installation.DNS).Info("Posting messages")
	if w.channelID == "" {
		w.Error = fmt.Errorf("failed to post message. You need to create channels first")
		return w
	}
	for i := 0; i < samples; i++ {
		_, response := w.mmClient.CreatePost(&mmodel.Post{
			UserId:    w.userID,
			ChannelId: w.channelID,
			Message:   "super big test",
		})
		if response.StatusCode != 201 {
			w.Error = fmt.Errorf("failed to post a message status = %d, message = %s", response.StatusCode, response.Error.Message)
			return w
		}
	}
	return w
}

// CreateTeam creates a team in the installation
func (w *TestWicker) CreateTeam() *TestWicker {
	w.logger.WithField("DNS", w.installation.DNS).Info("Creating a team")
	if w.userID == "" {
		w.Error = fmt.Errorf("failed to create team. You need to create a user first")
		return w
	}

	name := fmt.Sprintf("team-%d", rand.Intn(100))
	team, response := w.mmClient.CreateTeam(&mmodel.Team{
		Name:        name,
		DisplayName: name,
		Type:        mmodel.TEAM_OPEN,
	})
	if response.StatusCode != 201 {
		w.Error = fmt.Errorf("failed to create a team: status code = %d, message = %s", response.StatusCode, response.Error.Message)
		return w
	}
	w.teamID = team.Id
	return w
}

// AddTeamMember add the newly created user to3 a team in the installation
func (w *TestWicker) AddTeamMember() *TestWicker {
	w.logger.WithField("DNS", w.installation.DNS).Info("Adding team member")
	if w.teamID == "" {
		w.Error = fmt.Errorf("failed to add a team member. You need to create a team first")
		return w
	}

	_, response := w.mmClient.AddTeamMember(w.teamID, w.userID)
	if response.StatusCode != 201 {
		w.Error = fmt.Errorf("failed to add a team member: status code = %d, message = %s", response.StatusCode, response.Error.Message)
		return w
	}
	return w
}

// CreateChannel creates a channel so we can post couple of messages
func (w *TestWicker) CreateChannel() *TestWicker {
	w.logger.WithField("DNS", w.installation.DNS).Info("Creating channel")
	channel, response := w.mmClient.CreateChannel(&mmodel.Channel{
		CreatorId: w.userID,
		TeamId:    w.teamID,
		Type:      mmodel.CHANNEL_OPEN,
		Name:      fmt.Sprintf("channel-%d", rand.Intn(100)),
	})
	if response.StatusCode != 201 {
		w.Error = fmt.Errorf("failed to create channel status = %d, message = %s", response.StatusCode, response.Error.Message)
		w.logger.WithError(w.Error).Error()
		return w
	}
	w.channelID = channel.Id
	return w
}

// CreateIncomingWebhook creates an incoming webhook
func (w *TestWicker) CreateIncomingWebhook() *TestWicker {
	w.logger.WithField("DNS", w.installation.DNS).Info("Creating incoming webhook")
	if w.userID == "" {
		w.Error = fmt.Errorf("failed to post message. You need to create a user first")
		return w
	}
	if w.channelID == "" {
		w.Error = fmt.Errorf("failed to post message. You need to create channels first")
		return w
	}
	_, response := w.mmClient.CreateIncomingWebhook(&mmodel.IncomingWebhook{
		ChannelId: w.channelID,
		UserId:    w.userID,
	})
	if response.StatusCode != 201 {
		w.Error = fmt.Errorf("failed to create channel status = %d, message = %s", response.StatusCode, response.Error.Message)
		w.logger.WithError(w.Error).Error()
		return w
	}
	return w
}

// WaitForInstallationStable waits for installation to become stable
func (w *TestWicker) WaitForInstallationStable(ctx context.Context) *TestWicker {
	w.logger.WithField("DNS", w.installation.DNS).Info("Waiting installation")
	for {
		i, err := w.provisionerClient.GetInstallation(w.installation.ID, &cmodel.GetInstallationRequest{})
		if err != nil {
			w.Error = errors.Wrap(err, "client.GetInstallation")
			return w
		}
		w.logger.WithFields(logrus.Fields{
			"DNS":   w.installation.DNS,
			"State": w.installation.State,
		}).Info("Waiting installation")

		if i.State == cmodel.InstallationStateStable {
			w.installation = i
			return w
		}
		if i.State == cmodel.InstallationStateCreationFailed {
			w.installation = i
			w.Error = errors.Wrapf(err, "Installation creation failed with ID: %s", w.installation.ID)
			return w
		}

		select {
		case <-ctx.Done():
			w.Error = errors.New("timed out waiting to become stable")
			return w
		case <-time.After(10 * time.Second):
		}
		time.Sleep(10 * time.Second)
	}
}

// SetupInstallation creates a user and login with this one
func (w *TestWicker) SetupInstallation() *TestWicker {
	u := &mmodel.User{
		Username: "testwick",
		Email:    "testwick@example.mattermost.com",
		Password: "T3stw1ck10!",
	}
	_, response := w.mmClient.CreateUser(u)
	if response.StatusCode != 201 {
		w.Error = fmt.Errorf("failed to create admin user status = %d, message = %s", response.StatusCode, response.Error.Message)
		return w
	}
	w.mmClient.Logout()
	userLogged, response := w.mmClient.Login(u.Username, u.Password)
	if response.StatusCode != 200 {
		w.Error = fmt.Errorf("failed logging user: username = %s, status code = %d, message = %s", u.Username, response.StatusCode, response.Error.Message)
		return w
	}
	w.userID = userLogged.Id
	return w
}

// IsUP cpings the installation till return statusCode=200 and status=OK
func (w *TestWicker) IsUP(ctx context.Context) *TestWicker {
	w.mmClient = mmodel.NewAPIv4Client(fmt.Sprintf("https://%s", w.installation.DNS))
	for {
		status, response := w.mmClient.GetPing()
		w.logger.WithFields(logrus.Fields{
			"DNS":         w.installation.DNS,
			"Status":      status,
			"Status Code": response.StatusCode,
		}).Info("Pings installation")
		if response.StatusCode == 200 && status == "OK" {
			return w
		}

		select {
		case <-ctx.Done():
			w.Error = fmt.Errorf("failed to ping status = %d, message = %s", response.StatusCode, response.Error.Message)
			return w
		case <-time.After(10 * time.Second):
		}
	}
}
