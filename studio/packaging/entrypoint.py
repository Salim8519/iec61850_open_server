"""Frozen application entry point; no dependency installation at startup."""
from studio_app.launcher.entrypoint import main

if __name__ == "__main__":
    main()
