package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bootdotdev/gator/internal/config"
	"github.com/bootdotdev/gator/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	registeredCommands map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.registeredCommands[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	f, ok := c.registeredCommands[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return f(s, cmd)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("the login handler expects a single argument, the username")
	}

	username := cmd.args[0]
	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("user %s does not exist", username)
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("User has been set to %s\n", username)
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("could not reset users: %w", err)
	}

	fmt.Println("Database reset successfully!")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("the register handler expects a name argument")
	}

	name := cmd.args[0]
	now := time.Now().UTC()

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
	})
	if err != nil {
		return fmt.Errorf("could not create user: %w", err)
	}

	err = s.cfg.SetUser(user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("User created: %+v\n", user)
	return nil
}

func handlerAgg(s *state, cmd command) error {
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return fmt.Errorf("could not fetch feed: %w", err)
	}

	fmt.Printf("%+v\n", feed)
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("could not get users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerListFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("could not get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	for _, feed := range feeds {
		fmt.Printf("* Name: %s\n", feed.FeedName)
		fmt.Printf("* URL:  %s\n", feed.FeedUrl)
		fmt.Printf("* User: %s\n", feed.UserName)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("usage: %s <name> <url>", cmd.name)
	}

	feedName := cmd.args[0]
	feedURL := cmd.args[1]

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      feedName,
		Url:       feedURL,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("could not create feed: %w", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not follow feed: %w", err)
	}

	fmt.Println("Feed created and followed:")
	fmt.Printf("* ID:        %s\n", feed.ID)
	fmt.Printf("* Name:      %s\n", feed.Name)
	fmt.Printf("* URL:       %s\n", feed.Url)
	fmt.Printf("* UserID:    %s\n", feed.UserID)

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("usage: %s <url>", cmd.name)
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("feed not found: %w", err)
	}

	followRecord, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not follow feed: %w", err)
	}

	fmt.Printf("Feed: %s followed by User: %s\n", followRecord.FeedName, followRecord.UserName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("could not get follows: %w", err)
	}

	for _, f := range follows {
		fmt.Printf("* %s\n", f.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("usage: %s <url>", cmd.name)
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("feed not found: %w", err)
	}

	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not unfollow feed: %w", err)
	}

	return nil
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	programState := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	cmds := commands{
		registeredCommands: make(map[string]func(*state, command) error),
	}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("feeds", handlerListFeeds)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))

	if len(os.Args) < 2 {
		log.Fatalf("error: not enough arguments provided")
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	cmd := command{
		name: cmdName,
		args: cmdArgs,
	}

	err = cmds.run(programState, cmd)
	if err != nil {
		log.Fatalf("error running command: %v", err)
	}
}
