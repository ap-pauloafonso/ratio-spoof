package generator

type DefaultRoundingGenerator struct{}

func NewDefaultRoudingGenerator() (*DefaultRoundingGenerator, error) {
	return &DefaultRoundingGenerator{}, nil

}

func (d *DefaultRoundingGenerator) Round(downloadCandidateNextAmount, uploadCandidateNextAmount, leftCandidateNextAmount, pieceSize int) (downloaded, uploaded, left int) {

	down := downloadCandidateNextAmount
	up := uploadCandidateNextAmount - (uploadCandidateNextAmount % (16 * 1024))
	l := leftCandidateNextAmount - (leftCandidateNextAmount % pieceSize)
	return down, up, l
}
