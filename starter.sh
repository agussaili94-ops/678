#!/bin/bash

SESSION="bigo"
DIR="$(pwd)"

cd $DIR
tmux kill-session -t $SESSION 2>/dev/null

tmux new-session -d -s $SESSION -n "DASHBOARD"
tmux send-keys -t $SESSION "python 678_web.py" C-m

tmux split-window -v -t $SESSION
tmux send-keys -t $SESSION.1 "./678_multi" C-m

tmux set-option -t $SESSION mouse on
tmux select-pane -t $SESSION.1

clear
echo "=========================================="
echo "    678 WEB RECORDER BERHASIL DIMUAT      "
echo "    Web: http://localhost:8080           "
echo "=========================================="
sleep 1
tmux attach-session -t $SESSION
