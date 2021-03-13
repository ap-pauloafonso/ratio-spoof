package generator

type RoundingGenerator interface {
	Round(downloadCandidateNextAmount, uploadCandidateNextAmount, leftCandidateNextAmount, pieceSize int) (downloaded, uploaded, left int)
}

func NewRoundingGenerator(code string) (RoundingGenerator, error) {
	return &DefaultRoundingGenerator{}, nil

}

type DefaultRoundingGenerator struct{}

func (d *DefaultRoundingGenerator) Round(downloadCandidateNextAmount, uploadCandidateNextAmount, leftCandidateNextAmount, pieceSize int) (downloaded, uploaded, left int) {

	down := downloadCandidateNextAmount
	up := uploadCandidateNextAmount - (uploadCandidateNextAmount % (16 * 1024))
	l := leftCandidateNextAmount - (leftCandidateNextAmount % pieceSize)
	return down, up, l
}
