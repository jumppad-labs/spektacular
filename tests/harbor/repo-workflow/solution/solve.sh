#!/bin/bash
set -e

cd /app

spektacular init claude

# Start the guided add naming the repo up front, then walk every step,
# supplying at each one the answer a user would have agreed to.
spektacular repo new --data '{"location":"/srv/harbour"}'
spektacular repo goto --data '{"step":"name","name":"harbour"}'
spektacular repo goto --data '{"step":"description","description":"Tide-gate scheduling for dockside cranes"}'
spektacular repo goto --data '{"step":"role","role":"application"}'
spektacular repo goto --data '{"step":"tags","tags":["typescript","scheduling"]}'
spektacular repo goto --data '{"step":"placement","placement":"inside"}'
spektacular repo goto --data '{"step":"confirm"}'
spektacular repo goto --data '{"step":"register"}'
spektacular repo goto --data '{"step":"finished"}'

# This scripted walkthrough registers the repo, but it deliberately proves
# none of the conversational criteria. Whether each question arrived on its own
# turn carrying a proposal, and whether Spektacular's vocabulary stayed out of
# what the user was shown, are properties of a live agent's transcript and are
# checked only against a real run.
