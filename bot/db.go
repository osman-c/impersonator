package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbName = "admin"
const cName = "states"

var client *mongo.Client
var coll *mongo.Collection
var ctx context.Context

func getConnectionString() string {
	port := envLookup("DB_PORT")
	username := envLookup("DB_USERNAME")
	password := envLookup("DB_PASSWORD")
	return fmt.Sprintf("mongodb://%v:%v@mongo:%v", username, password, port)
}

func initializeDatabase() {
	uri := getConnectionString()
	ctx = context.TODO()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Panic(err)
	}

	client.Database(dbName).CreateCollection(ctx, cName)
	coll = client.Database(dbName).Collection(cName)
}

func initializeStates() {
	guilds, err := session.UserGuilds(100, "", "")
	if err != nil {
		log.Panicln(err)
	}

	for _, guild := range guilds {
		filter := bson.D{{"guildid", guild.ID}}

		err := coll.FindOne(ctx, filter)
		if err != nil {
			state := makeDefaultState(guild.ID)
			_, err := coll.InsertOne(ctx, *state)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func getState(guildID string) (State, error) {
	var state State
	filter := bson.D{{"guildid", guildID}}

	err := coll.FindOne(ctx, filter).Decode(&state)
	if err != nil {
		return State{}, err
	}
	return state, nil
}
