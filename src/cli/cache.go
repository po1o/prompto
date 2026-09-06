package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/daemon"
	"github.com/po1o/prompto/src/daemon/ipc"

	"github.com/spf13/cobra"
)

// sessionOnly limits `cache show` to the calling shell's own session.
var sessionOnly bool

// clearInit also removes the shell init scripts cached on disk.
var clearInit bool

// errNoDaemon is what every subcommand reports when it cannot reach the
// daemon. The caches live only in the daemon's memory, so without one there is
// nothing to read, clear or configure — and saying so beats printing an empty
// listing that looks like an answer.
const errNoDaemon = "no daemon running: the cache lives in the daemon, start a shell with prompto enabled first"

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Interact with the prompto cache",
	Long: `Interact with the prompto cache.

The cache lives in the daemon's memory and is never written to disk, so these
commands talk to the running daemon.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

var cachePathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the directory prompto writes generated files to",
	Long: `Print the directory prompto writes generated files to.

This holds the generated shell init scripts and, with --trace, the logs. The
cache itself is in-memory and has no path; use "prompto cache show" for that.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), cache.Path())
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Drop everything the daemon has cached",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		if clearInit {
			if err := cache.ClearInit(); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				exitcode = 1
				return
			}

			if !ipc.SocketExists() {
				fmt.Fprintln(cmd.OutOrStdout(), "init scripts cleared")
				return
			}
		}

		client, ok := cacheClient(cmd)
		if !ok {
			return
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), daemonCallTimeout)
		defer cancel()

		if err := client.CacheClear(ctx); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			exitcode = 1
			return
		}

		if clearInit {
			fmt.Fprintln(cmd.OutOrStdout(), "cache and init scripts cleared")
			return
		}

		fmt.Fprintln(cmd.OutOrStdout(), "cache cleared")
	},
}

var cacheTTLCmd = &cobra.Command{
	Use:   "ttl [days]",
	Short: "Get or set how long cached values live",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var days int
		if len(args) > 0 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "TTL must be a whole number of days: %v\n", err)
				exitcode = 2
				return
			}

			if parsed <= 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "TTL must be at least one day, got %d\n", parsed)
				exitcode = 2
				return
			}

			days = parsed
		}

		client, ok := cacheClient(cmd)
		if !ok {
			return
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), daemonCallTimeout)
		defer cancel()

		if len(args) == 0 {
			currentDays, err := client.CacheGetTTL(ctx)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				exitcode = 1
				return
			}

			fmt.Fprintf(cmd.OutOrStdout(), "TTL: %d days\n", currentDays)
			return
		}

		if err := client.CacheSetTTL(ctx, days); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			exitcode = 1
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "TTL set to %d days\n", days)
	},
}

var cacheShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print everything the daemon has cached",
	Long: `Print everything the daemon has cached.

Entries are grouped by the cache holding them: "device" and "session" are the
process-wide stores segments write to, "rendered segments" is cached prompt
output. Session-scoped rendered segments belong to a single shell and are
listed per session id, which is that shell's pid.

Values whose key names a credential are shown as <redacted>.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		client, ok := cacheClient(cmd)
		if !ok {
			return
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), daemonCallTimeout)
		defer cancel()

		var sessionID string
		if sessionOnly {
			sessionID = strconv.Itoa(os.Getppid())
		}

		response, err := client.CacheShow(ctx, sessionID)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			exitcode = 1
			return
		}

		writeCacheScopes(cmd.OutOrStdout(), response.Scopes)
	},
}

// cacheClient dials the running daemon. It deliberately does not start one:
// these commands inspect and mutate cache state, and a daemon started just to
// answer them would report an empty cache it had only just created.
func cacheClient(cmd *cobra.Command) (*daemon.Client, bool) {
	if !ipc.SocketExists() {
		fmt.Fprintln(cmd.ErrOrStderr(), errNoDaemon)
		exitcode = 1
		return nil, false
	}

	client, err := daemon.NewClient()
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), errNoDaemon)
		exitcode = 1
		return nil, false
	}

	return client, true
}

func writeCacheScopes(out io.Writer, scopes []*ipc.CacheScope) {
	var written int

	for _, scope := range scopes {
		if len(scope.Entries) == 0 {
			continue
		}

		if written > 0 {
			fmt.Fprintln(out)
		}
		written++

		fmt.Fprintf(out, "%s\n", cacheScopeTitle(scope))

		writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		for _, entry := range scope.Entries {
			fmt.Fprintf(writer, "  %s\t%s\t%s\n", entry.Key, cacheEntryValue(entry), cacheEntryLifetime(entry))
		}
		_ = writer.Flush()
	}

	if written == 0 {
		fmt.Fprintln(out, "the cache is empty")
	}
}

func cacheScopeTitle(scope *ipc.CacheScope) string {
	title := fmt.Sprintf("%s (%d)", scope.Name, len(scope.Entries))
	if scope.SessionId == "" {
		return title
	}

	return fmt.Sprintf("%s (%d) · session %s", scope.Name, len(scope.Entries), scope.SessionId)
}

func cacheEntryValue(entry *ipc.CacheEntry) string {
	if entry.Redacted {
		return "<redacted>"
	}

	return entry.Value
}

func cacheEntryLifetime(entry *ipc.CacheEntry) string {
	if entry.Expired {
		return "expired"
	}

	if entry.Expires == 0 {
		return "never expires"
	}

	return "expires " + time.Unix(entry.Expires, 0).Format("2006-01-02 15:04:05")
}

func init() {
	cacheShowCmd.Flags().BoolVarP(&sessionOnly, "session", "s", false, "only this shell's session")
	cacheClearCmd.Flags().BoolVar(&clearInit, "init", false, "also clear cached shell init scripts on disk")
	cacheCmd.AddCommand(cachePathCmd, cacheClearCmd, cacheTTLCmd, cacheShowCmd)
	RootCmd.AddCommand(cacheCmd)
}
