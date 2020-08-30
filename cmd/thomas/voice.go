package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/itfactory-tm/thomas-bot/pkg/discordha"

	"github.com/bwmarrin/discordgo"
	"github.com/itfactory-tm/thomas-bot/pkg/mixer"

	"github.com/kelseyhightower/envconfig"

	"github.com/spf13/cobra"
)

// TODO: automate these
const itfDiscord = "687565213943332875"
const audioChannel = "688370622228725848"

var audioConnected = false

func init() {
	rootCmd.AddCommand(NewVoiceCmd())
}

type voiceCmdOptions struct {
	Token         string
	EtcdEndpoints []string `envconfig:"ETCD_ENDPOINTS"`

	ha *discordha.HA
}

// NewVoiceCmd generates the `serve` command
func NewVoiceCmd() *cobra.Command {
	s := voiceCmdOptions{}
	c := &cobra.Command{
		Use:     "voice",
		Short:   "Run the voice server",
		Long:    `This is a separate instance for voice services. This can only run in a single replica`,
		RunE:    s.RunE,
		PreRunE: s.Validate,
	}

	// TODO: switch to viper
	err := envconfig.Process("thomasbot", &s)
	if err != nil {
		log.Fatalf("Error processing envvars: %q\n", err)
	}

	return c
}

func (v *voiceCmdOptions) Validate(cmd *cobra.Command, args []string) error {
	if v.Token == "" {
		return errors.New("No token specified")
	}

	return nil
}

func (v *voiceCmdOptions) RunE(cmd *cobra.Command, args []string) error {
	ctx := context.TODO()

	dg, err := discordgo.New("Bot " + v.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}

	v.ha, err = discordha.New(discordha.Config{
		Session:       dg,
		HA:            len(v.EtcdEndpoints) > 0,
		EtcdEndpoints: v.EtcdEndpoints,
		Context:       ctx,
	})
	if err != nil {
		return fmt.Errorf("error creating Discord HA: %w", err)
	}

	voiceQueueChan := v.ha.WatchVoiceCommands(ctx, audioChannel)

	for {
		q := <-voiceQueueChan
		if audioConnected {
			continue
		}
		connected := make(chan struct{})
		v.connectVoice(dg, connected)
		<-connected
		// send again for voice to pick up
		v.ha.SendVoiceCommand(audioChannel, q)
	}

	return nil
}

func (v *voiceCmdOptions) connectVoice(dg *discordgo.Session, connected chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if audioConnected {
		connected <- struct{}{}
		return
	}

	audioConnected = true
	voiceQueueChan := v.ha.WatchVoiceCommands(ctx, audioChannel)

	dgv, err := dg.ChannelVoiceJoin(itfDiscord, audioChannel, false, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	connected <- struct{}{}

	encoder := mixer.NewEncoder()
	encoder.VC = dgv
	go encoder.Run()

	doneChan := make(chan struct{})
	go func() {
		var i uint64
		for {
			select {
			case f := <-voiceQueueChan:
				log.Println(f)
				go encoder.Queue(uint64(i), f)
				i++
			case <-doneChan:
				return
			}
		}
	}()

	time.Sleep(5 * time.Second) // waiting for first audio
	for !encoder.HasFinishedAll() {
		time.Sleep(5 * time.Second)
	}

	// Close connections once all are played
	dgv.Disconnect()
	dgv.Close()
	encoder.Stop()
	audioConnected = false
	doneChan <- struct{}{}
}
