package members

// type Revealer struct {
// 	*gtk.Revealer
// 	Container *Container

// 	button  *controller.Button
// 	lastRev bool
// }

// func NewRevealer(c *Container) *Revealer {
// 	s, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
// 	s.Show()

// 	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
// 	b.Show()
// 	b.Add(s)
// 	b.Add(c)

// 	r, _ := gtk.RevealerNew()
// 	r.Add(b)
// 	r.Show()
// 	r.SetTransitionDuration(50)
// 	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_LEFT)

// 	return &Revealer{
// 		Revealer:  r,
// 		Container: c,
// 	}
// }

// func (r *Revealer) BindController(c *controller.Container) {
// 	r.button = c.Add("system-users-symbolic", r, r.GetRevealChild())
// 	r.button.Hide()
// }

// func (r *Revealer) SetRevealChild(reveal bool) {
// 	r.Revealer.SetRevealChild(reveal)
// 	r.button.SetActive(reveal)
// }

// func (r *Revealer) OnClick(b *controller.Button) {
// 	r.SetRevealChild(!r.GetRevealChild())
// }

// func (r *Revealer) Cleanup() {
// 	r.lastRev = r.GetRevealChild()

// 	r.SetRevealChild(false)
// 	r.button.SetActive(false)
// 	r.button.Hide()

// 	r.Container.Cleanup()
// }

// func (r *Revealer) LoadGuild(id discord.Snowflake) error {
// 	semaphore.IdleMust(func() {
// 		r.button.Show()
// 		r.button.SetActive(r.lastRev)
// 		r.SetRevealChild(r.lastRev)
// 	})

// 	return r.Container.LoadGuild(id)
// }
