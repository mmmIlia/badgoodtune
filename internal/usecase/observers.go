package usecase

type AuthObserver interface {
	UserRegistered()
	LoginSucceeded()
}

type AlbumObserver interface {
	AlbumCreated()
}

type ReviewObserver interface {
	ReviewCreated(score int)
}
