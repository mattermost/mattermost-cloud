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
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// MattermostRequester the interface which describes mattermost API client
type MattermostRequester interface {
	GetPing() (string, *mmodel.Response)
	Logout() (bool, *mmodel.Response)
	CreateTeam(team *mmodel.Team) (*mmodel.Team, *mmodel.Response)
	AddTeamMember(teamID, userID string) (*mmodel.TeamMember, *mmodel.Response)
	CreatePost(post *mmodel.Post) (*mmodel.Post, *mmodel.Response)
	CreateUser(user *mmodel.User) (*mmodel.User, *mmodel.Response)
	CreateChannel(channel *mmodel.Channel) (*mmodel.Channel, *mmodel.Response)
	CreateIncomingWebhook(webhoo *mmodel.IncomingWebhook) (*mmodel.IncomingWebhook, *mmodel.Response)
	Login(username, password string) (*mmodel.User, *mmodel.Response)
}

// ProvisionerRequester the interface which describes Provisioner API client
type ProvisionerRequester interface {
	CreateInstallation(request *cmodel.CreateInstallationRequest) (*cmodel.InstallationDTO, error)
	GetInstallation(id string, request *cmodel.GetInstallationRequest) (*cmodel.InstallationDTO, error)
	DeleteInstallation(id string) error
}

// TestWicker data struct for test wicker scenarios
type TestWicker struct {
	provisionerClient ProvisionerRequester
	mmClient          MattermostRequester
	logger            *logrus.Logger
	installation      *cmodel.InstallationDTO
	userID            string
	channelID         string
	teamID            string
}

// NewTestWicker creates a testwicker
func NewTestWicker(provisionerRequester ProvisionerRequester, requester MattermostRequester, logger *logrus.Logger) *TestWicker {
	return &TestWicker{
		logger:            logger,
		mmClient:          requester,
		provisionerClient: provisionerRequester,
	}
}

// CreateInstallation interacts with provisioner and uses the provided request
// to create a new installation
func (w *TestWicker) CreateInstallation(request *cmodel.CreateInstallationRequest) func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("DNS", request.DNS).Info("Creating installation")
		if err := request.Validate(); err != nil {
			return errors.Wrap(err, "CreateInstallationRequest.Validate")
		}
		i, err := w.provisionerClient.CreateInstallation(request)
		if err != nil {
			return errors.Wrap(err, "client.CreateInstallation")
		}
		w.installation = i
		return nil
	}
}

// DeleteInstallation interacts with provisioner and uses the provided request
// to delete an installation
func (w *TestWicker) DeleteInstallation() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("ID", w.installation.ID).Info("Deleting installation")
		err := w.provisionerClient.DeleteInstallation(w.installation.ID)
		if err != nil {
			return errors.Wrap(err, "client.DeleteInstallation")
		}
		return nil
	}
}

// PostMessage post messages to the different channels which are created
// automatically by the wicker
func (w *TestWicker) PostMessage(samples int) func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, _ context.Context) error {
		w.logger.WithField("DNS", w.installation.DNS).Info("Posting messages")
		if w.channelID == "" {
			return fmt.Errorf("failed to post message. You need to create channels first")
		}
		for i := 0; i < samples; i++ {
			_, response := w.mmClient.CreatePost(&mmodel.Post{
				UserId:    w.userID,
				ChannelId: w.channelID,
				Message:   "super big test",
			})
			if response.StatusCode != 201 {
				return fmt.Errorf("failed to post a message status = %d, message = %s", response.StatusCode, response.Error.Message)
			}
		}
		return nil
	}
}

// CreateTeam creates a team in the installation
func (w *TestWicker) CreateTeam() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("DNS", w.installation.DNS).Info("Creating a team")
		if w.userID == "" {
			return fmt.Errorf("failed to create team. You need to create a user first")
		}

		name := fmt.Sprintf("team-%d", rand.Intn(100))
		team, response := w.mmClient.CreateTeam(&mmodel.Team{
			Name:        name,
			DisplayName: name,
			Type:        mmodel.TEAM_OPEN,
		})
		if response.StatusCode != 201 {
			return fmt.Errorf("failed to create a team: status code = %d, message = %s", response.StatusCode, response.Error.Message)
		}
		w.teamID = team.Id
		return nil
	}
}

// AddTeamMember add the newly created user to a team in the installation
func (w *TestWicker) AddTeamMember() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("DNS", w.installation.DNS).Info("Adding team member")
		if w.teamID == "" {
			return fmt.Errorf("failed to add a team member. You need to create a team first")
		}

		_, response := w.mmClient.AddTeamMember(w.teamID, w.userID)
		if response.StatusCode != 201 {
			return fmt.Errorf("failed to add a team member: status code = %d, message = %s", response.StatusCode, response.Error.Message)
		}
		return nil
	}
}

// CreateChannel creates a channel so we can post couple of messages
func (w *TestWicker) CreateChannel() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("DNS", w.installation.DNS).Info("Creating channel")
		channel, response := w.mmClient.CreateChannel(&mmodel.Channel{
			CreatorId: w.userID,
			TeamId:    w.teamID,
			Type:      mmodel.CHANNEL_OPEN,
			Name:      namesgenerator.GetRandomName(5),
		})
		if response.StatusCode != 201 {
			return fmt.Errorf("failed to create channel status = %d, message = %s", response.StatusCode, response.Error.Message)
		}
		w.channelID = channel.Id
		return nil
	}
}

// CreateIncomingWebhook creates an incoming webhook
func (w *TestWicker) CreateIncomingWebhook() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("DNS", w.installation.DNS).Info("Creating incoming webhook")
		if w.userID == "" {
			return fmt.Errorf("failed to post message. You need to create a user first")
		}
		if w.channelID == "" {
			return fmt.Errorf("failed to post message. You need to create channels first")
		}
		_, response := w.mmClient.CreateIncomingWebhook(&mmodel.IncomingWebhook{
			ChannelId: w.channelID,
			UserId:    w.userID,
		})
		if response.StatusCode != 201 {
			return fmt.Errorf("failed to create channel status = %d, message = %s", response.StatusCode, response.Error.Message)
		}
		return nil
	}
}

// WaitForInstallationStable waits for installation to become stable
func (w *TestWicker) WaitForInstallationStable() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		w.logger.WithField("DNS", w.installation.DNS).Info("Waiting installation")
		for {
			i, err := w.provisionerClient.GetInstallation(w.installation.ID, &cmodel.GetInstallationRequest{})
			if err != nil {
				return errors.Wrap(err, "client.GetInstallation")
			}
			w.logger.WithFields(logrus.Fields{
				"DNS":   w.installation.DNS,
				"State": w.installation.State,
			}).Info("Waiting installation")

			if i.State == cmodel.InstallationStateStable {
				w.installation = i
				return nil
			}
			if i.State == cmodel.InstallationStateCreationFailed {
				w.installation = i
				return errors.Wrapf(err, "Installation creation failed with ID: %s", w.installation.ID)
			}

			select {
			case <-ctx.Done():
				return errors.New("timed out waiting to become stable")
			case <-time.After(10 * time.Second):
			}
			time.Sleep(10 * time.Second)
		}
	}
}

// SetupInstallation creates a user and login with this one
func (w *TestWicker) SetupInstallation() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		u := &mmodel.User{
			Username: "testwick",
			Email:    "testwick@example.mattermost.com",
			Password: "T3stw1ck10!",
		}
		_, response := w.mmClient.CreateUser(u)
		if response.StatusCode != 201 {
			return fmt.Errorf("failed to create admin user status = %d, message = %s", response.StatusCode, response.Error.Message)
		}
		w.mmClient.Logout()
		userLogged, response := w.mmClient.Login(u.Email, u.Password)
		if response.StatusCode != 200 {
			return fmt.Errorf("failed logging user: username = %s, status code = %d, message = %s", u.Username, response.StatusCode, response.Error.Message)
		}
		w.userID = userLogged.Id
		return nil
	}
}

// IsUP cpings the installation till return statusCode=200 and status=OK
func (w *TestWicker) IsUP() func(w *TestWicker, ctx context.Context) error {
	return func(w *TestWicker, ctx context.Context) error {
		for {
			status, response := w.mmClient.GetPing()
			w.logger.WithFields(logrus.Fields{
				"DNS":         w.installation.DNS,
				"Status":      status,
				"Status Code": response.StatusCode,
			}).Info("Pings installation")
			if response.StatusCode == 200 && status == "OK" {
				return nil
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("failed to ping status = %d, message = %s", response.StatusCode, response.Error.Message)
			case <-time.After(10 * time.Second):
			}
		}
	}
}
