package stager_test

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cloudfoundry/dropsonde/autowire/metrics"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/receptor/fake_receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/fake_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/cloudfoundry-incubator/stager/stager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stage", func() {
	var (
		stager                        Stager
		bbs                           *fake_bbs.FakeStagerBBS
		stagingRequest                cc_messages.StagingRequestFromCC
		downloadTailorAction          models.ExecutorAction
		downloadAppAction             models.ExecutorAction
		downloadFirstBuildpackAction  models.ExecutorAction
		downloadSecondBuildpackAction models.ExecutorAction
		downloadBuildArtifactsAction  models.ExecutorAction
		runAction                     models.ExecutorAction
		uploadDropletAction           models.ExecutorAction
		uploadBuildArtifactsAction    models.ExecutorAction
		config                        Config
		fakeDiegoAPIClient            *fake_receptor.FakeClient
		callbackURL                   string
	)

	BeforeEach(func() {
		fakeDiegoAPIClient = new(fake_receptor.FakeClient)
		bbs = &fake_bbs.FakeStagerBBS{}
		logger := lager.NewLogger("fakelogger")

		callbackURL = "http://the-stager.example.com"

		config = Config{
			Circuses: map[string]string{
				"penguin":                "penguin-compiler",
				"rabbit_hole":            "rabbit-hole-compiler",
				"compiler_with_full_url": "http://the-full-compiler-url",
				"compiler_with_bad_url":  "ftp://the-bad-compiler-url",
			},
			MinDiskMB:          2048,
			MinMemoryMB:        1024,
			MinFileDescriptors: 256,
		}

		stager = New(bbs, callbackURL, fakeDiegoAPIClient, logger, config)

		stagingRequest = cc_messages.StagingRequestFromCC{
			AppId:                          "bunny",
			TaskId:                         "hop",
			AppBitsDownloadUri:             "http://example-uri.com/bunny",
			BuildArtifactsCacheDownloadUri: "http://example-uri.com/bunny-droppings",
			BuildArtifactsCacheUploadUri:   "http://example-uri.com/bunny-uppings",
			DropletUploadUri:               "http://example-uri.com/droplet-upload",
			Stack:                          "rabbit_hole",
			FileDescriptors:                512,
			MemoryMB:                       2048,
			DiskMB:                         3072,
			Buildpacks: []cc_messages.Buildpack{
				{Name: "zfirst", Key: "zfirst-buildpack", Url: "first-buildpack-url"},
				{Name: "asecond", Key: "asecond-buildpack", Url: "second-buildpack-url"},
			},
			Environment: cc_messages.Environment{
				{"VCAP_APPLICATION", "foo"},
				{"VCAP_SERVICES", "bar"},
			},
		}

		downloadTailorAction = models.EmitProgressFor(
			models.ExecutorAction{
				models.DownloadAction{
					From:     "http://file-server.com/v1/static/rabbit-hole-compiler",
					To:       "/tmp/circus",
					CacheKey: "tailor-rabbit_hole",
				},
			},
			"",
			"",
			"Failed to Download Tailor",
		)

		downloadAppAction = models.EmitProgressFor(
			models.ExecutorAction{
				models.DownloadAction{
					From: "http://example-uri.com/bunny",
					To:   "/app",
				},
			},
			"",
			"Downloaded App Package",
			"Failed to Download App Package",
		)

		downloadFirstBuildpackAction = models.EmitProgressFor(
			models.ExecutorAction{
				models.DownloadAction{
					From:     "first-buildpack-url",
					To:       "/tmp/buildpacks/0fe7d5fc3f73b0ab8682a664da513fbd",
					CacheKey: "zfirst-buildpack",
				},
			},
			"",
			"Downloaded Buildpack: zfirst",
			"Failed to Download Buildpack: zfirst",
		)

		downloadSecondBuildpackAction = models.EmitProgressFor(
			models.ExecutorAction{
				models.DownloadAction{
					From:     "second-buildpack-url",
					To:       "/tmp/buildpacks/58015c32d26f0ad3418f87dd9bf47797",
					CacheKey: "asecond-buildpack",
				},
			},
			"",
			"Downloaded Buildpack: asecond",
			"Failed to Download Buildpack: asecond",
		)

		downloadBuildArtifactsAction = models.Try(
			models.EmitProgressFor(
				models.ExecutorAction{
					models.DownloadAction{
						From: "http://example-uri.com/bunny-droppings",
						To:   "/tmp/cache",
					},
				},
				"",
				"Downloaded Build Artifacts Cache",
				"No Build Artifacts Cache Found.  Proceeding...",
			),
		)

		fileDescriptorLimit := uint64(512)

		runAction = models.EmitProgressFor(
			models.ExecutorAction{
				models.RunAction{
					Path: "/tmp/circus/tailor",
					Args: []string{
						"-appDir=/app",
						"-buildArtifactsCacheDir=/tmp/cache",
						"-buildpackOrder=zfirst-buildpack,asecond-buildpack",
						"-buildpacksDir=/tmp/buildpacks",
						"-outputBuildArtifactsCache=/tmp/output-cache",
						"-outputDroplet=/tmp/droplet",
						"-outputMetadata=/tmp/result.json",
					},
					Env: []models.EnvironmentVariable{
						{"VCAP_APPLICATION", "foo"},
						{"VCAP_SERVICES", "bar"},
					},
					Timeout:        15 * time.Minute,
					ResourceLimits: models.ResourceLimits{Nofile: &fileDescriptorLimit},
				},
			},
			"Staging...",
			"Staging Complete",
			"Staging Failed",
		)

		uploadDropletAction = models.EmitProgressFor(
			models.ExecutorAction{
				models.UploadAction{
					From: "/tmp/droplet",
					To:   "http://file-server.com/v1/droplet/bunny?" + models.CcDropletUploadUriKey + "=http%3A%2F%2Fexample-uri.com%2Fdroplet-upload",
				},
			},
			"",
			"Droplet Uploaded",
			"Failed to Upload Droplet",
		)

		uploadBuildArtifactsAction = models.Try(
			models.EmitProgressFor(
				models.ExecutorAction{
					models.UploadAction{
						From: "/tmp/output-cache",
						To:   "http://file-server.com/v1/build_artifacts/bunny?" + models.CcBuildArtifactsUploadUriKey + "=http%3A%2F%2Fexample-uri.com%2Fbunny-uppings",
					},
				},
				"",
				"Uploaded Build Artifacts Cache",
				"Failed to Upload Build Artifacts Cache.  Proceeding...",
			),
		)
	})

	It("increments the counter to track arriving staging messages", func() {
		metricSender := fake.NewFakeMetricSender()
		metrics.Initialize(metricSender)
		err := stager.Stage(stagingRequest)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(metricSender.GetCounter("StagingRequestsReceived")).Should(Equal(uint64(1)))
	})

	Context("when file the server is available", func() {
		BeforeEach(func() {
			bbs.GetAvailableFileServerReturns("http://file-server.com/", nil)
		})

		It("creates a cf-app-staging Task with staging instructions", func() {
			err := stager.Stage(stagingRequest)
			Ω(err).ShouldNot(HaveOccurred())

			desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)

			Ω(desiredTask.Domain).To(Equal("cf-app-staging"))
			Ω(desiredTask.TaskGuid).To(Equal("bunny-hop"))
			Ω(desiredTask.Stack).To(Equal("rabbit_hole"))
			Ω(desiredTask.Log.Guid).To(Equal("bunny"))
			Ω(desiredTask.Log.SourceName).To(Equal("STG"))
			Ω(desiredTask.ResultFile).To(Equal("/tmp/result.json"))

			var annotation models.StagingTaskAnnotation

			err = json.Unmarshal([]byte(desiredTask.Annotation), &annotation)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(annotation).Should(Equal(models.StagingTaskAnnotation{
				AppId:  "bunny",
				TaskId: "hop",
			}))

			Ω(desiredTask.Actions).Should(Equal([]models.ExecutorAction{
				models.EmitProgressFor(
					models.Parallel(
						downloadTailorAction,
						downloadAppAction,
						downloadFirstBuildpackAction,
						downloadSecondBuildpackAction,
						downloadBuildArtifactsAction,
					),
					"Fetching app, buildpacks (zfirst, asecond), artifacts cache...",
					"Fetching complete",
					"Fetching failed",
				),
				runAction,
				models.EmitProgressFor(
					models.Parallel(
						uploadDropletAction,
						uploadBuildArtifactsAction,
					),
					"Uploading droplet, artifacts cache...",
					"Uploading complete",
					"Uploading failed",
				),
			}))

			Ω(desiredTask.MemoryMB).To(Equal(2048))
			Ω(desiredTask.DiskMB).To(Equal(3072))
			Ω(desiredTask.CPUWeight).To(Equal(StagingTaskCpuWeight))
		})

		It("gives the task a callback URL to call it back", func() {
			err := stager.Stage(stagingRequest)
			Ω(err).ShouldNot(HaveOccurred())

			desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)
			Ω(desiredTask.CompletionCallbackURL).Should(Equal(callbackURL))
		})

		Describe("resource limits", func() {
			Context("when the app's memory limit is less than the minimum memory", func() {
				BeforeEach(func() {
					stagingRequest.MemoryMB = 256
				})

				It("uses the minimum memory", func() {
					err := stager.Stage(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())

					desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)
					Ω(desiredTask.MemoryMB).Should(BeNumerically("==", config.MinMemoryMB))
				})
			})

			Context("when the app's disk limit is less than the minimum disk", func() {
				BeforeEach(func() {
					stagingRequest.DiskMB = 256
				})

				It("uses the minimum disk", func() {
					err := stager.Stage(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())

					desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)
					Ω(desiredTask.DiskMB).Should(BeNumerically("==", config.MinDiskMB))
				})
			})

			Context("when the app's memory limit is less than the minimum memory", func() {
				BeforeEach(func() {
					stagingRequest.FileDescriptors = 17
				})

				It("uses the minimum file descriptors", func() {
					err := stager.Stage(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())

					desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)

					runAction = models.EmitProgressFor(
						models.ExecutorAction{
							models.RunAction{
								Path: "/tmp/circus/tailor",
								Args: []string{
									"-appDir=/app",
									"-buildArtifactsCacheDir=/tmp/cache",
									"-buildpackOrder=zfirst-buildpack,asecond-buildpack",
									"-buildpacksDir=/tmp/buildpacks",
									"-outputBuildArtifactsCache=/tmp/output-cache",
									"-outputDroplet=/tmp/droplet",
									"-outputMetadata=/tmp/result.json",
								},
								Env: []models.EnvironmentVariable{
									{"VCAP_APPLICATION", "foo"},
									{"VCAP_SERVICES", "bar"},
								},
								Timeout:        15 * time.Minute,
								ResourceLimits: models.ResourceLimits{Nofile: &config.MinFileDescriptors},
							},
						},
						"Staging...",
						"Staging Complete",
						"Staging Failed",
					)

					Ω(desiredTask.Actions).Should(Equal([]models.ExecutorAction{
						models.EmitProgressFor(
							models.Parallel(
								downloadTailorAction,
								downloadAppAction,
								downloadFirstBuildpackAction,
								downloadSecondBuildpackAction,
								downloadBuildArtifactsAction,
							),
							"Fetching app, buildpacks (zfirst, asecond), artifacts cache...",
							"Fetching complete",
							"Fetching failed",
						),
						runAction,
						models.EmitProgressFor(
							models.Parallel(
								uploadDropletAction,
								uploadBuildArtifactsAction,
							),
							"Uploading droplet, artifacts cache...",
							"Uploading complete",
							"Uploading failed",
						),
					}))
				})
			})
		})

		Context("when build artifacts download uris are not provided", func() {
			BeforeEach(func() {
				stagingRequest.BuildArtifactsCacheDownloadUri = ""
			})

			It("does not instruct the executor to download the cache", func() {
				err := stager.Stage(stagingRequest)
				Ω(err).ShouldNot(HaveOccurred())

				desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)

				Ω(desiredTask.Actions).Should(Equal([]models.ExecutorAction{
					models.EmitProgressFor(
						models.Parallel(
							downloadTailorAction,
							downloadAppAction,
							downloadFirstBuildpackAction,
							downloadSecondBuildpackAction,
						),
						"Fetching app, buildpacks (zfirst, asecond)...",
						"Fetching complete",
						"Fetching failed",
					),
					runAction,
					models.EmitProgressFor(
						models.Parallel(
							uploadDropletAction,
							uploadBuildArtifactsAction,
						),
						"Uploading droplet, artifacts cache...",
						"Uploading complete",
						"Uploading failed",
					),
				}))
			})
		})

		Context("when no compiler is defined for the requested stack in stager configuration", func() {
			BeforeEach(func() {
				stagingRequest.Stack = "no_such_stack"
			})

			It("returns an error", func() {
				err := stager.Stage(stagingRequest)

				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(Equal("no compiler defined for requested stack"))
			})
		})

		Context("when the compiler for the requested stack is specified as a full URL", func() {
			BeforeEach(func() {
				stagingRequest.Stack = "compiler_with_full_url"
			})

			It("uses the full URL in the download tailor action", func() {
				err := stager.Stage(stagingRequest)
				Ω(err).ShouldNot(HaveOccurred())

				desiredTask := fakeDiegoAPIClient.CreateTaskArgsForCall(0)

				downloadAction := desiredTask.Actions[0].Action.(models.EmitProgressAction).Action.Action.(models.ParallelAction).Actions[0].Action.(models.EmitProgressAction).Action.Action.(models.DownloadAction)
				Ω(downloadAction.From).Should(Equal("http://the-full-compiler-url"))
			})
		})

		Context("when the compiler for the requested stack is specified as a full URL with an unexpected scheme", func() {
			BeforeEach(func() {
				stagingRequest.Stack = "compiler_with_bad_url"
			})

			It("returns an error", func() {
				err := stager.Stage(stagingRequest)
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("when build artifacts download url is not a valid url", func() {
			BeforeEach(func() {
				stagingRequest.BuildArtifactsCacheDownloadUri = "not-a-uri"
			})

			It("return a url parsing error", func() {
				err := stager.Stage(stagingRequest)

				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("invalid URI"))
			})
		})

		Context("when the task has already been created", func() {
			BeforeEach(func() {
				fakeDiegoAPIClient.CreateTaskReturns(receptor.Error{
					Type:    receptor.TaskGuidAlreadyExists,
					Message: "ok, this task already exists",
				})
			})

			It("does not raise an error", func() {
				err := stager.Stage(stagingRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when the API call fails", func() {
			desireErr := errors.New("Could not connect!")

			BeforeEach(func() {
				fakeDiegoAPIClient.CreateTaskReturns(desireErr)
			})

			It("returns an error", func() {
				err := stager.Stage(stagingRequest)
				Ω(err).Should(Equal(desireErr))
			})
		})
	})

	Context("when file server is not available", func() {
		BeforeEach(func() {
			bbs.GetAvailableFileServerReturns("http://file-server.com/", storeadapter.ErrorKeyNotFound)
		})

		It("should return an error", func() {
			err := stager.Stage(cc_messages.StagingRequestFromCC{
				AppId:                          "bunny",
				TaskId:                         "hop",
				AppBitsDownloadUri:             "http://example-uri.com/bunny",
				BuildArtifactsCacheDownloadUri: "http://example-uri.com/bunny-droppings",
				Stack:    "rabbit_hole",
				MemoryMB: 256,
				DiskMB:   1024,
			})

			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("no available file server present"))
		})
	})
})
