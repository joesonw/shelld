package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	var (
		pLogLevel    zapcore.Level = zapcore.InfoLevel
		pLogColor    bool
		pLogEncoding string
		pLogOutput   []string
		pLogError    []string
		pListen      string
	)

	cmd := &cobra.Command{
		Use:  "shelld [script file]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logConfig := zap.Config{
				Level:       zap.NewAtomicLevelAt(pLogLevel),
				Development: false,
				Encoding:    pLogEncoding,
				EncoderConfig: zapcore.EncoderConfig{
					TimeKey:        "ts",
					LevelKey:       "level",
					NameKey:        "logger",
					CallerKey:      "caller",
					FunctionKey:    zapcore.OmitKey,
					MessageKey:     "msg",
					StacktraceKey:  "stacktrace",
					LineEnding:     zapcore.DefaultLineEnding,
					EncodeTime:     zapcore.ISO8601TimeEncoder,
					EncodeDuration: zapcore.MillisDurationEncoder,
					EncodeCaller:   zapcore.ShortCallerEncoder,
				},
				OutputPaths:      pLogOutput,
				ErrorOutputPaths: pLogError,
			}
			if pLogColor {
				logConfig.EncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
			}

			logger, err := logConfig.Build()
			if err != nil {
				return err
			}

			pool := &sync.Pool{
				New: func() interface{} {
					return &bytes.Buffer{}
				},
			}

			logger.Info("server started")
			return http.ListenAndServe(pListen, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				logger := logger.With(zap.String("path", r.RequestURI), zap.String("method", r.Method))
				cmd := exec.CommandContext(r.Context(), args[0])
				cmd.Args = append(cmd.Args, r.Method)
				cmd.Args = append(cmd.Args, r.RequestURI)
				for k := range r.Header {
					cmd.Args = append(cmd.Args, k)
					cmd.Args = append(cmd.Args, r.Header.Get(k))
				}

				defer r.Body.Close()

				output := pool.Get().(*bytes.Buffer)
				output.Reset()

				cmd.Stdout = output
				cmd.Stderr = output
				cmd.Stdin = r.Body

				start := time.Now()
				if err := cmd.Run(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					logger.Error("unable to run command", zap.Error(err), zap.Duration("duration", time.Since(start)))
					return
				}
				logger.Info("success ", zap.Duration("duration", time.Since(start)))
				w.WriteHeader(http.StatusOK)
				_, _ = io.Copy(w, output)
			}))
		},
	}

	cmd.Flags().AddFlag(pflag.PFlagFromGoFlag(&flag.Flag{
		Name:     "log-level",
		Usage:    "set log level",
		Value:    &pLogLevel,
		DefValue: "info",
	}))
	cmd.Flags().StringVar(&pLogEncoding, "log-encoding", "console", "log encoding (json or console)")
	cmd.Flags().StringSliceVar(&pLogOutput, "log-output", []string{"stderr"}, "log output paths")
	cmd.Flags().StringSliceVar(&pLogOutput, "log-error", []string{"stderr"}, "log error output paths")
	cmd.Flags().StringVar(&pListen, "listen", ":8080", "http listen address")
	cmd.Flags().BoolVar(&pLogColor, "log-color", false, "log output with color")

	log.Fatal(cmd.Execute())
}
