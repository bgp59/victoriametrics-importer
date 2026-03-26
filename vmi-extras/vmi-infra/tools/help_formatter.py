import argparse
import os


class CustomWidthFormatter(argparse.RawDescriptionHelpFormatter):
    def __init__(self, prog, indent_increment=2, max_help_position=24, width=None):
        # Set a fixed width or detect terminal width
        if width is None:
            try:
                # Attempt to get terminal width, default to 80 if not available
                width = int(os.environ.get("COLUMNS", 80))
            except (TypeError, ValueError):
                width = 80  # Fallback if COLUMNS is not a valid integer
        super().__init__(
            prog,
            indent_increment=indent_increment,
            max_help_position=max_help_position,
            width=width,
        )
