# Periodic Table card generator
This is a script that makes all 118 elements of the periodic table in little cards.

You will need to specify a font file for this to work
## How to run

### Run from source

1. **Clone the repo**
   ```bash
   git clone https://github.com/Beijing-corn87/Periodic-table-generator.git
   ```
   Make sure you have [go installed](https://go.dev/dl/)
2. **Download a font file**
    I recomend getting a font file from [Google Fonts](https://fonts.google.com/).

    If you want to use the font I use it is called [Roboto](https://fonts.google.com/specimen/Roboto). Use the bold version for more clarity.
3. **Set your colours (optional)**
   The colours.json file comes preset with a list of colours that I used but you can set them to another hex code.
4. **Run the script**
   #### Flags you need to set:
   |  Flags   |                             Description                               |        Example        |
   | -------- | --------------------------------------------------------------------- | --------------------- |
   | ``-font``    | Sets the font file you will use                                       | -font Roboto-Bold.ttf |
   | ``-colours`` | Sets the .json file for colours                                       | -colours colours.json |
   | ``-outdir``  | Sets the output for the images                                        | -outdir elements      |
   | ``-height``  | Sets the height of the output image (will calculate width acordingly) | -height 600           |

   Your command will look somthing like this:
   ```bash
   go run main.go -font Roboto-Bold.tff -colours colours.json -outdir elements -height 600
   ```

### Run Binary
**TBD (or not )**
