"""Test-wide setup: the app's daily update check must not reach the internet during tests."""
import os

os.environ["SCADABENCH_UPDATE_CHECK"] = "off"
