package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-playground/validator"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	appusecases "github.com/sousair/apollo-cloud/internal/application/usecases"
	"github.com/sousair/apollo-cloud/internal/infra/providers"
	gormrepositories "github.com/sousair/apollo-cloud/internal/infra/repositories/gorm"
	gormmodels "github.com/sousair/apollo-cloud/internal/infra/repositories/gorm/models"
	"github.com/sousair/apollo-cloud/internal/infra/repositories/s3"
	httphandlers "github.com/sousair/apollo-cloud/internal/presentation/http/handlers"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Panic("Error loading .env file")
	}

	postgresConnectionURL := os.Getenv("POSTGRES_CONNECTION_URL")

	db, err := gorm.Open(postgres.Open(postgresConnectionURL), &gorm.Config{})

	if err != nil {
		fmt.Println("Error connecting to database")
		panic(err)
	}

	err = db.AutoMigrate(gormmodels.OwnerModel{}, gormmodels.AlbumModel{}, gormmodels.MusicModel{})

	if err != nil {
		fmt.Println("Error migrating models")
		panic(err)
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})

	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	var (
		privateBucketName = os.Getenv("AWS_S3_PRIVATE_BUCKET_NAME")
		publicBucketName  = os.Getenv("AWS_S3_PUBLIC_BUCKET_NAME")
	)

	uuidProvider := uuidv4.NewUuidV4Provider()
	ownerRepository := gormrepositories.NewGormOwnerRepository(db)
	fileRepository := s3.NewS3FileRepository(awsSession, privateBucketName, publicBucketName)
	musicRepository := gormrepositories.NewGormMusicRepository(db)

	createOwnerUsecase := appusecases.NewCreateOwnerUseCase(uuidProvider, ownerRepository)
	createMusicUsecase := appusecases.NewCreateMusicUsecase(uuidProvider, fileRepository, musicRepository)

	validator := validator.New()
	createOwnerHandler := httphandlers.NewCreateOwnerHttpHandler(validator, createOwnerUsecase)
	createMusicHandler := httphandlers.NewCreateMusicHttpHandler(validator, createMusicUsecase)

	e := echo.New()

	e.POST("/owners", createOwnerHandler.Handle)
	e.POST("/musics", createMusicHandler.Handle)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", os.Getenv("HTTP_PORT"))))
}
