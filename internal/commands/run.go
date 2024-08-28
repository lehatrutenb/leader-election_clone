package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/commands/cmdargs"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/depgraph"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/metrics"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/ticker"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func InitRunCommand() (cobra.Command, error) {
	cmdArgs := cmdargs.RunArgs{}
	cmd := cobra.Command{
		Use:   "run",
		Short: "Starts a leader election node",
		Long: `This command starts the leader election node that connects to zookeeper
		and starts to try to acquire leadership by creation of ephemeral node`,
		RunE: func(cmd *cobra.Command, _ []string) (returnErr error) {
			dg := depgraph.New()
			// zkConn, err := dg.GetZkConn() ??? не понимаю осмысленности запускать это тут
			logger, err := dg.GetLogger()
			if err != nil {
				return fmt.Errorf("get logger: %w", err)
			}
			logger.Info("args received", slog.String("servers", strings.Join(cmdArgs.ZookeeperServers, ", ")))

			ctx, cncl := context.WithCancelCause(cmd.Context())
			defer func() {
				if err == nil {
					cncl(nil)
				} else {
					cncl(fmt.Errorf("app shut down with error %s", returnErr.Error()))
				}
			}()
			eg, ctx := errgroup.WithContext(ctx)

			sigQuit := make(chan os.Signal, 2)
			signal.Notify(sigQuit, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				select {
				case <-sigQuit:
					logger.Info("app got signal to shut down")
					cncl(fmt.Errorf("app shut down by signal"))
				case <-ctx.Done():
					return
				}
			}()

			metrics := metrics.InitPrometheus(ctx, logger, eg)

			runner, err := dg.GetRunner(metrics)
			if err != nil {
				return fmt.Errorf("get runner: %w", err)
			}

			initState, err := dg.GetInitState(ticker.GetTicker(), cmdArgs)
			if err != nil {
				return fmt.Errorf("get init state: %w", err)
			}

			logger.Info("app started init state")
			err = runner.Run(ctx, initState)
			if err != nil {
				return fmt.Errorf("run states: %w", err)
			}

			if ctx.Err() != nil {
				_ = eg.Wait()
				return ctx.Err()
			}
			return eg.Wait()
		},
	}

	cmd.Flags().StringSliceVarP(&(cmdArgs.ZookeeperServers), "zk-servers", "s", []string{"zoo1:2181", "zoo2:2182", "zoo3:2183"}, "Set the zookeeper servers.")
	cmd.Flags().DurationVarP(&(cmdArgs.LeaderTimeout), "leader-timeout", "l", 300*time.Millisecond, "Set the leader file write timeout.")
	cmd.Flags().DurationVarP(&(cmdArgs.MaxDeadLeaderTimeout), "dead-leader-timeout", "t", 400*time.Millisecond, "Set the max timeout zookeper will wait for dead leader.")
	cmd.Flags().DurationVarP(&(cmdArgs.FailoverQuickRetryTimeout), "failover-quick-retry-timeout", "q", 50*time.Millisecond, "Set retry frequency to reconnect to zookeper as dead leader.")
	cmd.Flags().DurationVarP(&(cmdArgs.FailoverSlowRetryStep), "failover-slow-retry-step", "r", 500*time.Millisecond, "Set step timeout of retrying to return to attemper state .")
	cmd.Flags().DurationVarP(&(cmdArgs.FailoverMaxStateDuration), "failover-max-duration", "w", 10*time.Second, "Set max failover duration as a state.")
	cmd.Flags().DurationVarP(&(cmdArgs.AttempterTimeout), "attempter-timeout", "a", 300*time.Millisecond, "Set the attempt to become leader timeout.")
	cmd.Flags().StringVarP(&(cmdArgs.ElectionFileDir), "election-file-dir", "f", "/election", "Set the path to write file to become leader.")
	cmd.Flags().StringVarP(&(cmdArgs.LeaderFileDir), "leader-file-dir", "d", "/data", "Set the path to write files as leader.")
	cmd.Flags().IntVarP(&(cmdArgs.StorageCapacity), "storage-capacity", "c", 5, "Set max amount of files in leader dir.")

	return cmd, nil
}
