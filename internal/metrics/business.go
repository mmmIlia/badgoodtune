package metrics

func (m *Metrics) UserRegistered() { m.users.Inc() }

func (m *Metrics) LoginSucceeded() { m.logins.Inc() }

func (m *Metrics) AlbumCreated() { m.albums.Inc() }

func (m *Metrics) ReviewCreated(score int) {
	m.reviews.Inc()
	m.reviewScore.Observe(float64(score))
}
