package main

import (
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type Job struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
}

func doWork(job *Job, db *gorm.DB) {
	if job.Session.State.User.ID == job.Message.Author.ID {
		return
	}

	switch {
	case IsCommand("karma", job.Message.Content):
		CheckKarma(job, db)
	default:
		ModKarma(job, db)
	}
}

func worker(workQueue <-chan Job, cancel <-chan struct{}, db *gorm.DB) {
	for {
		select {
		case <-cancel:
			return
		case job := <-workQueue:
			doWork(&job, db)
		}
	}
}
