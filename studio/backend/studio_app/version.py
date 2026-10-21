"""Single source of the ScadaBench application version.

Development reports the current release with "+dev" (the update check then behaves
exactly as in the packaged app; a test keeps it equal to website/src/data/site.ts).
The protected build overwrites this file in its sealed source copy
(studio/tools/protected_source.py) with the real version passed to compile_protected.py.
"""
__version__ = "0.1.21+dev"
