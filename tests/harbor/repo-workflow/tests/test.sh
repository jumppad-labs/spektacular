#!/bin/bash

# Run the test suite
pytest /tests/test_repo_workflow.py -v

if [ $? -eq 0 ]; then
  echo 1 > /logs/verifier/reward.txt
else
  echo 0 > /logs/verifier/reward.txt
fi
