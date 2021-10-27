package window

// type Header struct {
// 	*handy.TitleBar
// 	Main *gtk.Stack
// }

// func initHeader() error {
// 	if Window.Header != nil {
// 		return nil
// 	}

// 	h := handy.NewTitleBar()
// 	h.Show()

// 	// // empty box for 0 width
// 	// b, err := gtk.NewBox(gtk.OrientationHorizontal, 0)
// 	// if err != nil {
// 	// 	return errors.Wrap(err, "Failed to create an empty box")
// 	// }
// 	// h.SetCustomTitle(b)

// 	// Main stack
// 	s := newStack()

// 	Window.Header = &Header{
// 		TitleBar: h,
// 		Main:     s,
// 	}
// 	h.Add(s)
// 	Window.Window.SetTitlebar(h)

// 	return nil
// }
