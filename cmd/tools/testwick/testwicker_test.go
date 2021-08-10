package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/cmd/tools/testwick/mocks"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	cmodel "github.com/mattermost/mattermost-cloud/model"
	mmodel "github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/assert"
)

func TestCreateInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)
	samples := []struct {
		description string
		err         error
		request     *cmodel.CreateInstallationRequest
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "dns validation error",
			request: &model.CreateInstallationRequest{
				OwnerID: "testwicker",
			},
			setup: func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "DNS")
			},
		},
		{
			description: "cluster size error",
			request: &model.CreateInstallationRequest{
				DNS:     "test.cloud.com",
				OwnerID: "testwicker",
			},
			setup: func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "invalid cluster size")
			},
		},
		{
			description: "provisioner api error",
			request: &model.CreateInstallationRequest{
				DNS:       "test.cloud.com",
				Size:      "100users",
				Affinity:  cmodel.InstallationAffinityMultiTenant,
				Database:  cmodel.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
				Filestore: cmodel.InstallationFilestoreBifrost,
				OwnerID:   "testwicker",
			},
			setup: func(wicker *TestWicker) {

				provisionerRequester.EXPECT().
					CreateInstallation(gomock.Any()).
					Return(&model.InstallationDTO{}, errors.New("some provisioner error"))
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "provisioner error")
			},
		},
		{
			description: "create installation success",
			request: &model.CreateInstallationRequest{
				DNS:       "test.cloud.com",
				Size:      "100users",
				Affinity:  cmodel.InstallationAffinityMultiTenant,
				Database:  cmodel.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
				Filestore: cmodel.InstallationFilestoreBifrost,
				OwnerID:   "testwicker",
			},
			setup: func(wicker *TestWicker) {
				provisionerRequester.EXPECT().
					CreateInstallation(gomock.Any()).
					Return(&model.InstallationDTO{
						Installation: &cmodel.Installation{
							ID:  "123",
							DNS: "test.cloud.com",
						},
					}, nil)
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, wicker.installation.ID)
				assert.NotEmpty(t, wicker.installation.DNS)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			v.setup(wicker)

			f := wicker.CreateInstallation(v.request)
			err := f(wicker, context.TODO())

			v.expected(wicker, err)
		})
	}
}

func TestWaitForInstallationStable(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	samples := []struct {
		description string
		ctx         func() context.Context
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "provisioner API error",
			ctx: func() context.Context {
				return context.TODO()
			},
			setup: func(wicker *TestWicker) {
				provisionerRequester.EXPECT().
					GetInstallation(gomock.Any(), &cmodel.GetInstallationRequest{}).
					Return(nil, errors.New("some provisioner error"))
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			description: "provisioner installation state failed",
			ctx: func() context.Context {
				return context.TODO()
			},
			setup: func(wicker *TestWicker) {
				wicker.installation.State = cmodel.InstallationStateCreationFailed
				provisionerRequester.EXPECT().
					GetInstallation(gomock.Any(), &cmodel.GetInstallationRequest{}).
					Return(wicker.installation, nil)
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed installation creation")
			},
		},
		{
			description: "context timed out",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
				return ctx
			},
			setup: func(wicker *TestWicker) {
				provisionerRequester.EXPECT().
					GetInstallation(gomock.Any(), &cmodel.GetInstallationRequest{}).
					Return(wicker.installation, nil)
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "timed out waiting to become stable")
			},
		},
		{
			description: "installation stable success",
			ctx: func() context.Context {
				return context.TODO()
			},
			setup: func(wicker *TestWicker) {
				wicker.installation.State = cmodel.InstallationStateStable
				provisionerRequester.EXPECT().
					GetInstallation(gomock.Any(), &cmodel.GetInstallationRequest{}).
					Return(wicker.installation, nil)
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.WaitForInstallationStable()
			err := f(wicker, v.ctx())

			v.expected(wicker, err)
		})
	}
}

func TestDeleteInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	t.Run("provisioner API error", func(t *testing.T) {
		provisionerRequester.EXPECT().
			DeleteInstallation(gomock.Any()).
			Return(errors.New("some provisioner error"))

		wicker.installation = &model.InstallationDTO{
			Installation: &cmodel.Installation{
				ID: "123",
			},
		}
		f := wicker.DeleteInstallation()
		err := f(wicker, context.TODO())

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "provisioner error")
	})

	t.Run("delete success", func(t *testing.T) {
		provisionerRequester.EXPECT().
			DeleteInstallation(gomock.Any()).
			Return(nil)

		wicker.installation = &model.InstallationDTO{
			Installation: &cmodel.Installation{
				ID: "123",
			},
		}
		f := wicker.DeleteInstallation()
		err := f(wicker, context.TODO())

		assert.NoError(t, err)
	})
}

func TestPostMessage(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)
	samples := []struct {
		description string
		err         error
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "channel ID is missing",
			setup:       func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create channels first")
			},
		},
		{
			description: "create api post error",
			setup: func(wicker *TestWicker) {
				wicker.channelID = "1234"
				mmRequester.EXPECT().
					CreatePost(gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed to post a message")
			},
		},
		{
			description: "create post error",
			setup: func(wicker *TestWicker) {
				wicker.channelID = "1234"
				mmRequester.EXPECT().
					CreatePost(gomock.Any()).Return(&mmodel.Post{}, &mmodel.Response{
					StatusCode: 201,
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.PostMessage(1)
			err := f(wicker, context.TODO())

			v.expected(wicker, err)
		})
	}
}

func TestCreateTeam(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	samples := []struct {
		description string
		err         error
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "user ID is missing",
			setup:       func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create a user first")
			},
		},
		{
			description: "create api team error",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				wicker.installation = &model.InstallationDTO{
					Installation: &cmodel.Installation{
						ID: "123",
					},
				}
				mmRequester.EXPECT().
					CreateTeam(gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed to create a team")
			},
		},
		{
			description: "create team success",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				wicker.installation = &model.InstallationDTO{
					Installation: &cmodel.Installation{
						ID: "123",
					},
				}
				mmRequester.EXPECT().
					CreateTeam(gomock.Any()).Return(&mmodel.Team{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 201,
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
				assert.NotEmpty(t, wicker.teamID)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.CreateTeam()
			err := f(wicker, context.TODO())

			v.expected(wicker, err)
		})
	}
}

func TestAddTeamMember(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	samples := []struct {
		description string
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "team ID is missing",
			setup:       func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create a team and a user first")
			},
		},
		{
			description: "user ID is missing",
			setup:       func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				wicker.teamID = "1234"
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create a team and a user first")
			},
		},
		{
			description: "add api team member error",
			setup: func(wicker *TestWicker) {
				wicker.teamID = "1234"
				mmRequester.EXPECT().
					AddTeamMember(gomock.Any(), gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed to add a team member")
			},
		},
		{
			description: "add team member success",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				mmRequester.EXPECT().
					AddTeamMember(wicker.teamID, wicker.userID).Return(&mmodel.TeamMember{
					TeamId: wicker.teamID,
					UserId: wicker.userID,
				}, &mmodel.Response{
					StatusCode: 201,
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.AddTeamMember()
			err := f(wicker, context.TODO())

			v.expected(wicker, err)
		})
	}
}

func TestCreateChannel(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	samples := []struct {
		description string
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "user ID is missing",
			setup:       func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create a user first")
			},
		},
		{
			description: "create api channel error",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				mmRequester.EXPECT().
					CreateChannel(gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed to create channel")
			},
		},
		{
			description: "create channel success",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				mmRequester.EXPECT().
					CreateChannel(gomock.Any()).Return(&mmodel.Channel{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 201,
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.CreateChannel()
			err := f(wicker, context.TODO())

			v.expected(wicker, err)

		})
	}
}

func TestCreateIncomingWebhook(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	samples := []struct {
		description string
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "user ID is missing",
			setup:       func(wicker *TestWicker) {},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create a user first")
			},
		},
		{
			description: "channel ID is missing",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "You need to create a channel first")
			},
		},
		{
			description: "create api incoming webhook error",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				wicker.channelID = "1234"
				mmRequester.EXPECT().
					CreateIncomingWebhook(gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed to create incoming webhook")
			},
		},
		{
			description: "create incoming webhook success",
			setup: func(wicker *TestWicker) {
				wicker.userID = "1234"
				wicker.channelID = "1234"
				mmRequester.EXPECT().
					CreateIncomingWebhook(gomock.Any()).Return(&mmodel.IncomingWebhook{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 201,
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.CreateIncomingWebhook()
			err := f(wicker, context.TODO())

			v.expected(wicker, err)
		})
	}
}

func TestSetupInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	mmRequester := mocks.NewMockMattermostRequester(gmctrl)
	provisionerRequester := mocks.NewMockProvisionerRequester(gmctrl)
	wicker := NewTestWicker(provisionerRequester, mmRequester, logger)

	samples := []struct {
		description string
		setup       func(wicker *TestWicker)
		expected    func(wicker *TestWicker, err error)
	}{
		{
			description: "create user api error",
			setup: func(wicker *TestWicker) {
				mmRequester.EXPECT().CreateUser(gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed to create admin user status")
			},
		},
		{
			description: "logout api error",
			setup: func(wicker *TestWicker) {
				mmRequester.EXPECT().CreateUser(gomock.Any()).Return(&mmodel.User{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 201,
				})
				mmRequester.EXPECT().Logout().Return(false, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed logged out user")
			},
		},
		{
			description: "login api error",
			setup: func(wicker *TestWicker) {
				mmRequester.EXPECT().CreateUser(gomock.Any()).Return(&mmodel.User{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 201,
				})
				mmRequester.EXPECT().Logout().Return(true, &mmodel.Response{
					StatusCode: 200,
				})
				mmRequester.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, &mmodel.Response{
					StatusCode: 400,
					Error:      &mmodel.AppError{Message: "any mattermost API error"},
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "failed logging user")
			},
		},
		{
			description: "login api success",
			setup: func(wicker *TestWicker) {
				mmRequester.EXPECT().CreateUser(gomock.Any()).Return(&mmodel.User{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 201,
				})
				mmRequester.EXPECT().Logout().Return(true, &mmodel.Response{
					StatusCode: 200,
				})
				mmRequester.EXPECT().Login(gomock.Any(), gomock.Any()).Return(&mmodel.User{
					Id: "1234",
				}, &mmodel.Response{
					StatusCode: 200,
				})
			},
			expected: func(wicker *TestWicker, err error) {
				assert.Nil(t, err)
				assert.NotEmpty(t, wicker.userID)
			},
		},
	}

	for _, v := range samples {
		t.Run(v.description, func(t *testing.T) {
			wicker.installation = &model.InstallationDTO{
				Installation: &cmodel.Installation{
					ID: "123",
				},
			}
			v.setup(wicker)

			f := wicker.SetupInstallation()
			err := f(wicker, context.TODO())

			v.expected(wicker, err)
		})
	}
}
