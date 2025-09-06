# Berkeley Mono font

This app is configured to use the "Berkeley Mono" typeface. To embed the font in the packaged app, place your licensed WOFF2 files in this folder with the following filenames:

- berkeley-mono-regular.woff2
- berkeley-mono-italic.woff2
- berkeley-mono-bold.woff2
- berkeley-mono-bold-italic.woff2

The CSS in `src/style.css` defines `@font-face` rules for these names and will automatically pick them up. If the files are not present, the app will fall back to system monospace fonts.

Note: Ensure you have the appropriate license to redistribute these font files with your application.
