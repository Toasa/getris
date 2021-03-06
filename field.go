package main

import (
    "fmt"
    "sort"
    "github.com/veandco/go-sdl2/sdl"
)

type Board [FIELD_HEIGHT][FIELD_WIDTH]*Cell

type Field struct {
    surface *sdl.Surface
    board Board
    curMino *Mino
    dropPredMino *Mino
}

func newField(window *sdl.Window) (*Field, error) {
    var board Board
    for h := 0; h < FIELD_HEIGHT; h++ {
        for w := 0; w < FIELD_WIDTH; w++ {
            board[h][w] = newCell(VOID, colorVOID)
        }
    }

	surface, err := window.GetSurface()
    if err != nil {
        return nil, err
    }

    f := &Field{
        board: board,
        surface: surface,
    }

    return f, nil
}

func (f *Field) draw() {
    getRect := func (h, w int) sdl.Rect{
        return sdl.Rect {
            int32(w * CELL_LEN),
            int32(h * CELL_LEN),
            CELL_LEN,
            CELL_LEN,
        }
    }

    for h := 0; h < FIELD_HEIGHT; h++ {
        for w := 0; w < FIELD_WIDTH; w++ {
            cell := f.board[h][w]
            rect := getRect(h, w)
            f.surface.FillRect(&rect, uint32(cell.color))
        }
    }
}

func (f *Field) setCell(h, w int, c *Cell) {
    f.board[h][w] = c
}

func (f *Field) getCell(h, w int) *Cell {
    return f.board[h][w]
}

func (f *Field) setMino(m *Mino, s State) {
    if m == nil {
        return
    }

    var c Color
    if s == Falling {
        c = m.color()
        f.curMino = m
    } else if s == DropPred {
        c = colorDropPred
        f.dropPredMino = m
    } else if s == Fixed {
        c = m.color()
        f.curMino = nil
        f.dropPredMino = nil
    } else {
        panic(fmt.Sprintf("Unknown state: %d", s))
    }

    for _, coord := range m.coords {
        if coord.isExceedTop() {
            continue
        }

        h := coord.getHeight()
        w := coord.getWidth()

        cell := newCell(s, c)
        f.setCell(h, w, cell)
    }
}

func (f *Field) addMino(m *Mino) {
    f.setDropPredMino(m)
    f.setMino(m, Falling)
    f.draw()
}

func (f *Field) isGameOver(m *Mino) bool {
    coords := m.coords
    for _, coord := range coords {
        if coord.isExceedTop() {
            continue
        }

        h := coord.getHeight()
        w := coord.getWidth()
        cell := f.getCell(h, w)
        if cell.state == Fixed {
            return true
        }
    }
    return false
}

func (f *Field) moveMino(dst *Mino, src Mino) {
    f.blank(src)
    f.setMino(dst, Falling)
}

// Hard drop is implemented by repeating single down
// as continuously as possible.
func (f *Field) getHardDropMino(m *Mino) *Mino {
    new_m := m
    prev_m := f.curMino

    for !f.atBottom(*new_m) {
        prev_m = new_m
        new_m = new_m.move(MoveDrop)
    }

    return prev_m
}

func (f *Field) setDropPredMino(m *Mino) {
    new_m := f.getHardDropMino(m)
    if f.dropPredMino != nil {
        f.blank(*(f.dropPredMino))
    }
    f.setMino(new_m, DropPred)
}

// attempt attempts to a specified move for current tetri-mino.
// The move may fail, for exapmle, in the case of go out the field
// or overlap the already fixed cell. When move fails, then we do nothing.
func (f *Field) attempt(move Move) {
    var new_m *Mino

    if move == MoveHardDrop {
        new_m = f.getHardDropMino(f.curMino)
        f.moveMino(new_m, *(f.curMino))
    } else {
        new_m = f.curMino.move(move)
    }

    // current tetri-mino reaches to bottom or
    // already fixed cells.
    if (move == MoveDrop && f.atBottom(*new_m)) ||
    (move == MoveHardDrop) {
        f.setMino(f.curMino, Fixed)
        f.draw()
        return
    }

    if !f.legalMove(*new_m) {
        // If move is not rotation, moving attempt simply is failed.
        if !(move == RotLeft || move == RotRight) {
            return
        }

        // If the control reaches here, assume that move is rotaion.
        // Currently, rotation of mino failed and
        // we try to slide rotated mino horizontally.
        //
        // For a explanation, we introduce following three figures.
        // 
        //     |                |                |
        //     |■               | ■              |  ■
        //     |■              □|■■              |■■■
        //     |■■              |                |
        //     |                |                |
        //     +-----           +-----           +-----
        //      Fig1             Fig2             Fig3
        //
        // First let we are in Fig1 case and we will rotate mino left.
        // Then, in the naive implement, we are fall into the Fig2 case,
        // and rotation failed. However, such a situation does not occur
        // with ordinaty tetris. Instead slide the rotated mino to the
        // right, like a in Fig3.

        const (
            exceedLeft uint8 = iota
            exceedRight
            noExceedSide
        )

        // check a new mino is exceed sideways.
        // If so, checkExceedSide reports which side and
        // the number that need to slide.
        checkExceedSide := func(b Board, m Mino) (uint8, int) {
            exLeftFlag := false
            exRightFlag := false
            for _, coord := range m.coords {
                if coord.isExceedTop() {
                    continue
                }
                w := coord.getWidth()
                if w < 0 {
                    exLeftFlag = true
                    break
                }
                if w >= FIELD_WIDTH {
                    exRightFlag = true
                    break
                }
            }

            if !exLeftFlag && !exRightFlag {
                return noExceedSide, 0
            }

            // To avoid multiple counting exceeded cells
            // which have same width and different height.
            hMap := make(map[int]bool)
            if exLeftFlag {
                for _, coord := range m.coords {
                    w := coord.getWidth()
                    if w < 0 {
                        hMap[coord.getHeight()] = true
                    }
                }
                return exceedLeft, len(hMap)
            } else {
                for _, coord := range m.coords {
                    w := coord.getWidth()
                    if w >= FIELD_WIDTH {
                        hMap[coord.getHeight()] = true
                    }
                }
                return exceedRight, len(hMap)
            }
        }
        ex, n := checkExceedSide(f.board, *new_m)
        if ex == noExceedSide {
            return
        }

        if ex == exceedLeft {
            for i := 0; i < n; i++ {
                new_m = new_m.move(MoveRight)
            }
        } else if ex == exceedRight {
            for i := 0; i < n; i++ {
                new_m = new_m.move(MoveLeft)
            }
        }

        // Attempt to slide mino failed.
        if !f.legalMove(*new_m) {
            return
        }

        // If execution reaches here, slide of rotated mino succeeded.
    }

    f.setDropPredMino(new_m)
    f.moveMino(new_m, *(f.curMino))
    f.draw()
}

func (f *Field) attemptDescent() {
    lines := f.getCompleteHorizontalLines()
    if len(lines) > 0 {
        f.eraseLines(lines)
        f.dropRemains(lines)
    }
}

func (f *Field) legalMove(m Mino) bool {
    coords := m.coords
    for _, coord := range coords {
        if coord.isExceedSide() || coord.isExceedBottom() {
            return false
        }

        if coord.isExceedTop() {
            continue
        }

        h := coord.getHeight()
        w := coord.getWidth()

        cell := f.getCell(h, w)
        if cell.state == Fixed {
            return false
        }
    }
    return true
}

func (f *Field) blank(m Mino) {
    coords := m.coords
    for _, coord := range coords {
        if coord.isExceedTop() {
            continue
        }

        h := coord.getHeight()
        w := coord.getWidth()
        cell := f.getCell(h, w)
        cell.toVoid()
    }
}

func (f *Field) atBottom(m Mino) bool {
    coords := m.coords
    for _, coord := range coords {
        if coord.isExceedTop() {
            continue
        }

        h := coord.getHeight()
        w := coord.getWidth()
        if h >= FIELD_HEIGHT {
            return true
        }

        cell := f.getCell(h, w)
        if cell.state == Fixed {
            return true
        }
    }

    return false
}

func (f *Field) getCompleteHorizontalLines() []int {
    lines := []int{}
    for h := 0; h < FIELD_HEIGHT; h++ {
        all_fixed := true
        for w := 0; w < FIELD_WIDTH; w++ {
            if f.getCell(h, w).state != Fixed {
                all_fixed = false
            }
        }
        if all_fixed {
            lines = append(lines, h)
        }
    }
    return lines
}

func (f *Field) eraseLine(h int) {
    for w := 0; w < FIELD_WIDTH; w++ {
        cell := f.getCell(h, w)
        cell.toVoid()
    }
}

func (f *Field) eraseLines(lines []int) {
    for _, l := range lines {
        f.eraseLine(l)
    }
}

func (f *Field) dropRemains(erasedLines []int) {
    f.eraseLines(erasedLines)

    if len(erasedLines) == 0 {
        return
    }

    erasedMap := func(eLines []int) []bool {
        eMap := make([]bool, FIELD_HEIGHT, FIELD_HEIGHT)
        for _, h := range eLines {
            eMap[h] = true
        }
        return eMap
    }(erasedLines)

    nFixedLine := func(eMap []bool) int {
        n := 0
        for h := FIELD_HEIGHT-1; h >= 0; h-- {
            if eMap[h] {
                break
            }
            n++
        }
        return n
    }(erasedMap)

    // Do nothing.
    if nFixedLine >= FIELD_HEIGHT {
        return
    }

    copyLine := func(dstL, srcL int) {
        for w := 0; w < FIELD_WIDTH; w++ {
            f.board[dstL][w] = f.board[srcL][w]
        }
    }

    copySrcL := func(nFixed int, eMap []bool) int {
        for h := FIELD_HEIGHT-(1+nFixed); h >= 0; h-- {
            if !eMap[h] {
                return h
            }
        }
        return FIELD_HEIGHT
    }(nFixedLine, erasedMap)

    // No drop and erase lines only.
    if copySrcL >= FIELD_HEIGHT {
        return
    }

    sort.Sort(sort.IntSlice(erasedLines))
    copyDstL := erasedLines[len(erasedLines)-1]

    if copySrcL >= copyDstL {
        panic("Fail to drop line: copying")
    }

    // Assume that copySrcL is greater than copyDstL.
    for copySrcL >= 0 && copyDstL >= 0 {
        if copySrcL < 0{
            f.eraseLine(copyDstL)
        }
        copyLine(copyDstL, copySrcL)
        copyDstL--; copySrcL--
        for copySrcL >= 0 && erasedMap[copySrcL] {
            copySrcL--
        }

        for copySrcL < 0 && copyDstL >= 0 {
            f.eraseLine(copyDstL)
            copyDstL--
        }
    }

    f.draw()
}
