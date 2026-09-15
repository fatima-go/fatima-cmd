package deployui

import (
	"context"
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"github.com/fatima-go/fatima-opm/lifecycle"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatima-go/fatima-opm/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func RunJSON(c *Client, o Options) (resultErr error) {
	defer func() { resultErr = share.UnsupportedRPC(resultErr, "Jupiter 또는 대상 Juno") }()
	parent, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()
	var result proto.Message
	var err error
	switch o.Command {
	case "upload":
		if o.RequestID == "" {
			return fmt.Errorf("--request-id is required for an upload in JSON mode")
		}
		result, err = c.Upload(ctx, o.Value, o.RequestID, nil)
	case "", "artifacts":
		result, err = c.Artifacts(ctx)
	case "rollouts":
		result, err = c.Rollouts(ctx)
	case "watch":
		result, err = c.Get(ctx, o.Value)
	case "create":
		if o.RequestID == "" {
			return fmt.Errorf("--request-id is required for create in JSON mode")
		}
		result, err = c.Create(ctx, &api.CreateRollout{RequestId: o.RequestID, ArtifactId: o.ArtifactID, Group: o.Group, FirstPackageId: o.First})
		if err == nil {
			p := result.(*api.Rollout)
			// JSON create owns the rollout until it finishes or the process exits.
			// Emit snapshots so another authorized CLI can approve WAITING work.
			for {
				b, e := protojson.MarshalOptions{UseProtoNames: true}.Marshal(p)
				if e != nil {
					return e
				}
				if _, e = fmt.Fprintln(os.Stdout, string(b)); e != nil {
					return e
				}
				if lifecycle.Terminal(p.State) {
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
				}
				next, e := c.Get(ctx, p.Id)
				if e != nil {
					continue
				}
				p = next
			}
		}
	case "act":
		result, err = c.Act(ctx, &api.RolloutAction{Id: o.Value, Action: o.Action, ExpectedRevision: o.Revision})
	default:
		return fmt.Errorf("unknown command")
	}
	if err != nil {
		return err
	}
	b, err := protojson.MarshalOptions{Indent: "  ", UseProtoNames: true}.Marshal(result)
	if err == nil {
		_, err = fmt.Fprintln(os.Stdout, string(b))
	}
	return err
}
