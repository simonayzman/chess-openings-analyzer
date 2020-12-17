# Code to Measure time taken by program to execute. 
import time 
import os
import matplotlib.pyplot as plt
from sys import argv
from subprocess import run, DEVNULL

saveFileName = 'speedup_graph.png'
runCount = 20
if len(argv) > 1:
    try:
        candidateRunCount = int(argv[1])
        runCount = candidateRunCount
        print(f"[Run count adjusted to {runCount}]\n")
    except:
        pass

if len(argv) > 2:
    try:
        saveFileName = argv[2]
        print(f"[Speedup graph will saved to testing/{saveFileName}]\n")
    except:
        pass

orderedInputs = ["1", "2", "3"]
orderedThreads = [1, 2, 4, 6, 8]
orderedColors = ['red', 'green', 'blue']

appName = "chess-openings-app"
projectDirectory = os.getcwd()
srcDirectory = f"{projectDirectory}/src"
appDirectory = f"{srcDirectory}/app"
testingDirectory = f"{projectDirectory}/testing"

os.chdir(appDirectory)
run(["go", "build", "-o", f"{srcDirectory}/{appName}", "app.go"])
os.chdir(srcDirectory)

print(f"SEQUENTIAL\n===========\n")
sequentialOutputs = {}
serialSumTimes = 0
for runId in range(runCount):
    begin = time.time() 
    result = run([f"{srcDirectory}/{appName}", "process"], stdout=DEVNULL, stderr=DEVNULL)
    end = time.time() 
    elapsed = end - begin
    serialSumTimes += elapsed
    print(f"Run {runId + 1}:\t\t{round(elapsed, 3)} seconds")
serialAverageTime = float(serialSumTimes) / float(runCount)
print(f"Average:\t{round(serialAverageTime, 3)} seconds\n")

print(f"PARALLEL\n===========\n")
speedups = []
for threadCount in orderedThreads:        
    parallelSumTimes = 0
    for runId in range(runCount):
        begin = time.time() 
        result = run([f"{srcDirectory}/{appName}", "process", str(threadCount)], stdout=DEVNULL, stderr=DEVNULL)
        end = time.time() 
        elapsed = end - begin
        parallelSumTimes += elapsed
        print(f"Run {runId + 1} ({threadCount} threads):\t{round(elapsed, 3)} seconds")
    parallelAverageTime = float(parallelSumTimes) / float(runCount)
    speedup = serialAverageTime / parallelAverageTime
    speedups.append(speedup)
    print(f"Average:\t\t{round(parallelAverageTime, 3)} seconds\n")

os.chdir(testingDirectory)
plt.plot(orderedThreads, speedups)
plt.xlabel('Number of Threads (N)')
plt.ylabel('Speedup')
plt.title('Number of Threads vs Speedup')
plt.grid(True)
plt.savefig(f"graphs/{saveFileName}")
