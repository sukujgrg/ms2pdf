package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/sukujgrg/ms2pdf/internal/auth"
	"github.com/sukujgrg/ms2pdf/internal/config"
	"github.com/sukujgrg/ms2pdf/internal/filetype"
	"github.com/sukujgrg/ms2pdf/internal/graph"
)

func New() *cli.Command {
	appID := &cli.StringFlag{
		Name:    "app-id",
		Usage:   "Entra application (client) ID",
		Sources: cli.EnvVars("MS2PDF_CLIENT_ID"),
	}
	tenant := &cli.StringFlag{
		Name:    "tenant",
		Usage:   "optional Entra directory ID; omit for any personal or work account (/common)",
		Sources: cli.EnvVars("MS2PDF_TENANT_ID"),
	}
	return &cli.Command{
		Name:  "ms2pdf",
		Usage: "Convert Office documents to PDF via Microsoft Graph",
		Flags: []cli.Flag{appID, tenant},
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Sign in via the system browser (any personal or work Microsoft account)",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "device-code", Usage: "use device code instead of a browser (work/school accounts in --tenant only)"},
				},
				Action: loginAction,
			},
			{
				Name:   "logout",
				Usage:  "Forget cached Microsoft accounts",
				Action: logoutAction,
			},
			{
				Name:   "whoami",
				Usage:  "Print the cached signed-in account",
				Action: whoamiAction,
			},
			{
				Name:      "convert",
				Usage:     "Upload one file, convert it to PDF, then delete the temp OneDrive item",
				ArgsUsage: "<file>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "output PDF path"},
					&cli.StringFlag{Name: "type", Usage: "override input type when the suffix is missing or wrong"},
				},
				Action: convertAction,
			},
		},
	}
}

func loginAction(ctx context.Context, cmd *cli.Command) error {
	id, tenant, err := resolveIDs(cmd)
	if err != nil {
		return err
	}
	sess, err := auth.Open(ctx, id, tenant)
	if err != nil {
		return err
	}
	var acct auth.Account
	if cmd.Bool("device-code") {
		acct, err = sess.LoginDeviceCode(ctx, cmd.Root().Writer)
	} else {
		acct, err = sess.Login(ctx, cmd.Root().Writer)
	}
	if err != nil {
		return err
	}
	if err := persistLogin(id, tenant); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.Root().Writer, "logged in as %s\n", acct.Username)
	return err
}

func logoutAction(ctx context.Context, cmd *cli.Command) error {
	sess, err := openSession(ctx, cmd)
	if err != nil {
		return err
	}
	if err := sess.Logout(ctx); err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Root().Writer, "logged out")
	return err
}

func whoamiAction(ctx context.Context, cmd *cli.Command) error {
	sess, err := openSession(ctx, cmd)
	if err != nil {
		return err
	}
	acct, err := sess.WhoAmI(ctx)
	if err != nil {
		return notLoggedIn(err)
	}
	if acct.Name != "" && acct.Name != acct.Username {
		_, err = fmt.Fprintf(cmd.Root().Writer, "%s (%s)\n", acct.Username, acct.Name)
		return err
	}
	_, err = fmt.Fprintln(cmd.Root().Writer, acct.Username)
	return err
}

func convertAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() != 1 {
		return fmt.Errorf("usage: ms2pdf convert <file> [-o out.pdf] [--type ext]")
	}
	input := cmd.Args().First()
	ext, err := filetype.Resolve(input, cmd.String("type"))
	if err != nil {
		return err
	}
	output := cmd.String("output")
	if output == "" {
		output = filetype.DefaultOutput(input)
	}
	sess, err := openSession(ctx, cmd)
	if err != nil {
		return err
	}
	token, err := sess.SilentToken(ctx)
	if err != nil {
		return notLoggedIn(err)
	}
	if err := graph.New(token).ConvertFile(ctx, input, output, ext); err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Root().Writer, output)
	return err
}

func openSession(ctx context.Context, cmd *cli.Command) (*auth.Session, error) {
	id, tenant, err := resolveIDs(cmd)
	if err != nil {
		return nil, err
	}
	return auth.Open(ctx, id, tenant)
}

func resolveIDs(cmd *cli.Command) (string, string, error) {
	id, err := config.ResolveClientID(cmd.String("app-id"))
	if err != nil {
		return "", "", err
	}
	tenant, err := config.ResolveTenant(cmd.String("tenant"))
	if err != nil {
		return "", "", err
	}
	return id, tenant, nil
}

func persistLogin(id, tenant string) error {
	f, err := config.Load()
	if err != nil {
		return err
	}
	f.ClientID = id
	f.TenantID = tenant
	return config.Save(f)
}

func notLoggedIn(err error) error {
	if errors.Is(err, auth.ErrNotLoggedIn) {
		return fmt.Errorf("%w", auth.ErrNotLoggedIn)
	}
	return err
}
